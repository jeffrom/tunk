package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sosedoff/gitkit"
)

var ciSourceDir = Path("testdata/ci_mode")

type ciModeTestCase struct {
	name   string
	passwd string
	gitCfg *gitkit.Config
	// preOps happens before starting the server. for setup
	preOps []testOperation
	// srvOps happens on the server, directly in the bare repo, *after* preOps
	srvOps  []testOperation
	ops     []testOperation
	environ []string
	gitPath string
}

func strs(args ...string) []string { return args }

func TestTunkCIMode(t *testing.T) {
	if testing.Short() {
		t.Skip("-short")
	}
	if runtime.GOOS == "windows" {
		t.Skip("windows not supported (gitkit uses syscall.Kill)")
	}

	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	call(context.Background(), t, gitPath, "--version")

	tcs := []ciModeTestCase{
		{
			gitPath: gitPath,
			name:    "basic",
			passwd:  "coolpass",
			preOps: []testOperation{
				{Commit: "initial commit"},
				{Tag: "v0.1.0"},
				{GitArgs: strs("push", "--follow-tags", "origin", "master")},
			},
			ops: []testOperation{
				{Commit: "feat: a"},
				{GitArgs: strs("push", "origin", "master")},
				{TunkArgs: strs("--ci")},
			},
			environ: strs("GIT_TOKEN=coolpass"),
		},

		{
			gitPath: gitPath,
			name:    "main-branch",
			passwd:  "coolpass",
			srvOps: []testOperation{
				{GitArgs: strs("symbolic-ref", "refs/remotes/origin/HEAD", "refs/heads/main")},
				{GitArgs: strs("symbolic-ref", "HEAD", "refs/heads/main")},
			},
			preOps: []testOperation{
				{Commit: "initial commit"},
				{GitArgs: strs("branch", "-m", "master", "main")},
				{Tag: "v0.1.0"},
				{GitArgs: strs("push", "--follow-tags", "origin", "main")},
			},
			ops: []testOperation{
				{Commit: "feat: a"},
				{GitArgs: strs("push", "-u", "origin", "main")},
				{TunkArgs: strs("--ci")},
			},
			environ: strs("GIT_TOKEN=coolpass"),
		},

		{
			gitPath: gitPath,
			name:    "not-trunk",
			passwd:  "coolpass",
			ops: []testOperation{
				{Commit: "feat: a"},
				{GitArgs: strs("push", "-u", "origin", "master")},
				{GitArgs: strs("checkout", "-b", "coolbranch")},
				{Commit: "fix: b"},
				{GitArgs: strs("push", "-u", "origin", "coolbranch")},
				{TunkArgs: strs("--ci"), ShouldFail: true},
			},
			environ: strs("GIT_TOKEN=coolpass"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, runCITest(tc))
	}
}

func runCITest(tc ciModeTestCase) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		repoPath, err := os.MkdirTemp("", "tunk-repo")
		die(err)
		t.Logf("Clone dir: %s", repoPath)
		defer cleanupTempdir(t, repoPath)

		wd, err := os.Getwd()
		die(err)
		defer os.Chdir(wd)

		// setup env
		currEnv := os.Environ()
		defer resetEnviron(t, currEnv)
		os.Clearenv()
		for _, env := range tc.environ {
			parts := strings.SplitN(env, "=", 2)
			key, val := parts[0], parts[1]
			die(os.Setenv(key, val))
		}
		// make sure git is in path if path is unset
		if path := os.Getenv("PATH"); path == "" {
			gitDir, _ := filepath.Split(filepath.Clean(tc.gitPath))
			os.Setenv("PATH", gitDir)
		}

		srv := newGitServer(tc.passwd, tc.gitCfg)
		srvRepoDir := filepath.Join(srv.dir, "myrepo.git")
		die(os.MkdirAll(srvRepoDir, 0755))
		die(os.Chdir(srvRepoDir))
		call(ctx, t, "git", "init", "--bare", ".")

		srvRepoCloneDir, err := os.MkdirTemp("", "tunk-repo")
		die(err)
		die(os.Chdir(srvRepoCloneDir))
		call(ctx, t, "git", "clone", srvRepoDir, ".")
		for _, op := range tc.preOps {
			runOp(ctx, t, op)
		}
		// call(ctx, t, "git", "push", "--follow-tags", "origin", "master")

		die(os.Chdir(srvRepoDir))
		t.Logf("running %d bare repo ops", len(tc.srvOps))
		for _, op := range tc.srvOps {
			runOp(ctx, t, op)
		}

		addr := srv.start(t)
		defer srv.stop(t)

		cloneURL := fmt.Sprintf("http://%s/myrepo.git", addr)
		call(ctx, t, "git", "clone", cloneURL, repoPath)
		die(os.Chdir(repoPath))

		for _, op := range tc.ops {
			runOp(ctx, t, op)
		}

		// check results in "remote"
		die(os.Chdir(filepath.Join(srv.dir, "myrepo.git")))
		logOut := goldenGitLog(ctx, t, true)
		goldenPath := filepath.Join(ciSourceDir, tc.name, "expect")
		if env := goldenEnv; env != "" {
			t.Logf("Writing golden file at %s", goldenPath)
			dir, _ := filepath.Split(filepath.Clean(goldenPath))
			die(os.MkdirAll(dir, 0755))
			die(os.WriteFile(goldenPath, logOut, 0644))
			return
		}

		// compare goldenfile to output
		expectb, err := os.ReadFile(goldenPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Fatalf("No goldenfile at %s. Create one by rerunning with GOLDEN=1", goldenPath)
			}
			die(err)
		}

		if !bytes.Equal(expectb, logOut) {
			t.Fatalf("golden file didn't match. Either fix, or run: GOLDEN=1 go test on this test\n\nexpected:\n\n%s\n\ngot:\n\n%s", string(expectb), string(logOut))
		}
	}
}

type gitServer struct {
	cfg    gitkit.Config
	dir    string
	passwd string
	svc    *gitkit.Server
	http   *httptest.Server
}

func newGitServer(passwd string, cfg *gitkit.Config) *gitServer {
	dir, err := os.MkdirTemp("", "tunk-test")
	if err != nil {
		panic(err)
	}

	if cfg == nil {
		auth := false
		if passwd != "" {
			auth = true
		}
		cfg = &gitkit.Config{
			Dir:        dir,
			AutoCreate: true,
			AutoHooks:  true,
			Auth:       auth,
			Hooks: &gitkit.HookScripts{
				PreReceive: `echo "pre-receive"`,
			},
		}
	}

	return &gitServer{
		dir:    dir,
		passwd: passwd,
		cfg:    *cfg,
		svc:    gitkit.New(*cfg),
	}
}

func (g *gitServer) setup(t *testing.T) {
	t.Helper()
	t.Log("Setting up git server...")
	if g.passwd != "" {
		t.Logf("Using password: %q", g.passwd)
		g.svc.AuthFunc = func(cred gitkit.Credential, req *gitkit.Request) (bool, error) {
			t.Logf("auth attempt with password: %q", cred.Password)
			return cred.Password == g.passwd, nil
		}
	}
	if err := g.svc.Setup(); err != nil {
		t.Fatal(err)
	}
}

func (g *gitServer) start(t *testing.T) net.Addr {
	t.Helper()
	g.setup(t)
	g.http = httptest.NewUnstartedServer(g.svc)
	g.http.Start()
	addr := g.http.Listener.Addr()
	t.Logf("Test git server listening: %s", addr)
	return addr
}

func (g *gitServer) stop(t *testing.T) {
	t.Logf("Stopping git server and removing tmpdir %s", g.dir)
	g.http.Close()
	if t.Failed() {
		t.Logf("Test failed so leaving tmpdir in place: %s", g.dir)
		return
	}
	os.RemoveAll(g.dir)
}
