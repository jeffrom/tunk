package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ghodss/yaml"

	"github.com/jeffrom/tunk/vcs/gitcli"
)

var goldenEnv = os.Getenv("GOLDEN")

type testOperation struct {
	Commit   string   `json:"commit,omitempty"`
	Tag      string   `json:"tag,omitempty"`
	TunkArgs []string `json:"tunk,omitempty"`
	GitArgs  []string `json:"git,omitempty"`
}

type defaultModeTestCase struct {
	name    string
	dir     string
	environ []string
	gitPath string
}

func TestTunkDefaultMode(t *testing.T) {
	if testing.Short() {
		t.Skip("-short")
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}

	validRoot := Path("testdata/valid")
	validDirs, err := ioutil.ReadDir(validRoot)
	die(err)

	for _, dir := range validDirs {
		name := dir.Name()
		sourceDir := filepath.Join(validRoot, name)
		tc := defaultModeTestCase{name: name, dir: sourceDir, gitPath: gitPath}
		t.Run(name, runDefaultModeTest(tc))
	}
}

func runDefaultModeTest(tc defaultModeTestCase) func(t *testing.T) {
	return func(t *testing.T) {
		name := tc.name
		ctx := context.Background()
		currDir, err := os.Getwd()
		die(err)
		defer os.Chdir(currDir)

		tmpDir, err := ioutil.TempDir("", fmt.Sprintf("tunk-%s", name))
		die(err)
		defer func() {
			if t.Failed() {
				t.Logf("Test failed. Leaving temp dir: %s", tmpDir)
				return
			}
			t.Logf("Removing temp dir: %s", tmpDir)
			os.RemoveAll(tmpDir)
		}()

		die(os.Chdir(tmpDir))

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

		tunkYAML, err := ioutil.ReadFile(filepath.Join(tc.dir, "tunk.yaml"))
		if err == nil {
			die(ioutil.WriteFile(filepath.Join(tmpDir, "tunk.yaml"), tunkYAML, 0644))
		}
		call(ctx, t, "git", "init")
		call(ctx, t, "git", "config", "--local", "user.email", "tunk-test@example.com")
		call(ctx, t, "git", "config", "--local", "user.name", "tunk-test")

		testOpData, err := ioutil.ReadFile(filepath.Join(tc.dir, "test.yaml"))
		die(err)
		testopParts := bytes.Split(testOpData, []byte("---\n"))
		var testops []*testOperation
		for _, testopPart := range testopParts {
			testopPart = bytes.TrimSpace(testopPart)
			if len(testopPart) == 0 {
				continue
			}
			testop := &testOperation{}
			die(yaml.Unmarshal(testopPart, &testop))
			testops = append(testops, testop)
		}

		for _, testop := range testops {
			// fmt.Printf("op: %+v\n", testop)
			// fmt.Println(testop.TunkArgs == nil, len(testop.TunkArgs))
			runOp(ctx, t, *testop)
		}

		logOut := goldenGitLog(ctx, t)

		goldenPath := filepath.Join(tc.dir, "expect")
		if env := goldenEnv; env != "" {
			t.Logf("Writing golden file at %s", goldenPath)
			die(ioutil.WriteFile(goldenPath, logOut, 0644))
			return
		}

		// compare goldenfile to output
		expectb, err := ioutil.ReadFile(goldenPath)
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

func goldenGitLog(ctx context.Context, t testing.TB) []byte {
	t.Helper()
	logOut, err := exec.CommandContext(ctx,
		"git", "log", "--graph",
		"--pretty=format:%d %s",
		"--abbrev-commit",
	).Output()
	die(err)
	return logOut
}

func runOp(ctx context.Context, t *testing.T, testop testOperation) {
	t.Helper()
	if testop.Commit != "" {
		call(ctx, t, "git", "commit", "--allow-empty", "-m", testop.Commit)
	}
	if testop.Tag != "" {
		call(ctx, t, "git", "tag", "-a", testop.Tag, "-m", testop.Tag)
	}
	if testop.TunkArgs != nil {
		callTunk(t, testop.TunkArgs...)
	}
	if testop.GitArgs != nil {
		call(ctx, t, "git", testop.GitArgs...)
	}
}

func Path(p string) string {
	dir, err := findGoMod()
	die(err)

	finalPath := filepath.Join(dir, p)
	return finalPath
}

var gomodPath string

func findGoMod() (string, error) {
	if gomodPath != "" {
		return gomodPath, nil
	}

	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return "", errors.New("failed to get path of caller's file")
	}
	dir, _ := filepath.Split(file)

	for d := dir; d != "/"; d, _ = filepath.Split(filepath.Clean(d)) {
		gomodPath := filepath.Join(d, "go.mod")
		if _, err := os.Stat(gomodPath); err != nil {
			continue
		}
		gomodPath = d
		return d, nil
	}
	return "", errors.New("failed to find project root")
}

func call(ctx context.Context, t *testing.T, arg string, args ...string) {
	t.Helper()
	t.Logf("+ %s %s", arg, gitcli.ArgsString(args))

	var askpass string
	if arg == "git" {
		ap, cleanup, err := gitcli.SetupCreds()
		die(err)
		defer cleanup()
		askpass = ap
	}

	cmd := exec.CommandContext(ctx, arg, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if arg == "git" {
		cmd.Env = []string{
			"GIT_AUTHOR_NAME=tunk-test",
			"GIT_AUTHOR_EMAIL=tunk-test@example.com",
			"GIT_COMMITTER_NAME=tunk-test",
			"GIT_COMMITTER_EMAIL=tunk-test@example.com",
			fmt.Sprintf("GIT_ASKPASS=%s", askpass),
		}
		switch args[0] {
		case "config":
		default:
			cmd.Env = append(cmd.Env, "GIT_CONFIG=")
		}
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
}

func callTunk(t *testing.T, args ...string) {
	t.Helper()
	t.Logf("tunk(%s)", gitcli.ArgsString(args))
	if err := run(append([]string{"tunk"}, args...)); err != nil {
		t.Fatal(err)
	}
}

func resetEnviron(t testing.TB, environ []string) {
	t.Helper()
	t.Log("Resetting environment")

	os.Clearenv()
	for _, env := range environ {
		parts := strings.SplitN(env, "=", 2)
		key, val := parts[0], parts[1]
		die(os.Setenv(key, val))
	}
}
