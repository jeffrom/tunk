package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
	Commit     string   `json:"commit,omitempty"`
	Tag        string   `json:"tag,omitempty"`
	TunkArgs   []string `json:"tunk,omitempty"`
	GitArgs    []string `json:"git,omitempty"`
	ShouldFail bool     `json:"should_fail,omitempty"`
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
	call(context.Background(), t, gitPath, "--version")

	validRoot := Path("testdata/valid")
	validDirs, err := os.ReadDir(validRoot)
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

		tmpDir, err := os.MkdirTemp("", fmt.Sprintf("tunk-%s", name))
		die(err)
		defer cleanupTempdir(t, tmpDir)
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

		tunkYAML, err := os.ReadFile(filepath.Join(tc.dir, "tunk.yaml"))
		if err == nil {
			die(os.WriteFile(filepath.Join(tmpDir, "tunk.yaml"), tunkYAML, 0644))
		}
		call(ctx, t, "git", "init")
		call(ctx, t, "git", "config", "--local", "user.email", "tunk-test@example.com")
		call(ctx, t, "git", "config", "--local", "user.name", "tunk-test")

		testOpData, err := os.ReadFile(filepath.Join(tc.dir, "test.yaml"))
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

		logOut := goldenGitLog(ctx, t, false)

		goldenPath := filepath.Join(tc.dir, "expect")
		if env := goldenEnv; env != "" {
			t.Logf("Writing golden file at %s", goldenPath)
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
			// t.Fatalf("golden file didn't match. Either fix, or run: GOLDEN=1 go test on this test\n\nexpected:\n\n%s\n\ngot:\n\n%s", string(expectb), string(logOut))
			logDiff(t, expectb, logOut)
			t.Fatal("golden file didn't match. Either fix, or run: GOLDEN=1 go test on this test")
		}

	}
}

func goldenGitLog(ctx context.Context, t testing.TB, author bool) []byte {
	t.Helper()
	args := []string{"log", "--graph",
		"--abbrev-commit",
	}
	if author {
		args = append(args, "--pretty=format:%d %s \"%an\" <%ae>")
	} else {
		args = append(args, "--pretty=format:%d %s")
	}

	logOut, err := exec.CommandContext(ctx,
		"git", args...).Output()
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
		args := testop.TunkArgs
		t.Logf("tunk(%s)", gitcli.ArgsString(args))
		if err := run(append([]string{"tunk"}, args...)); !testop.ShouldFail && err != nil {
			t.Fatal(err)
		} else if testop.ShouldFail && err == nil {
			t.Fatal("expected error but got none")
		}
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

// func callTunk(t *testing.T, args ...string) {
// 	t.Helper()
// 	t.Logf("tunk(%s)", gitcli.ArgsString(args))
// 	if err := run(append([]string{"tunk"}, args...)); err != nil {
// 		t.Fatal(err)
// 	}
// }

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

func cleanupTempdir(t testing.TB, p string) {
	t.Helper()
	if t.Failed() {
		t.Logf("Test failed, leaving tempdir in place: %s", p)
		return
	}
	t.Logf("Removing tempdir %s", p)
	os.RemoveAll(p)
}

func logDiff(t testing.TB, expect, got []byte) {
	t.Helper()
	expectf, err := os.CreateTemp("", "tunk-diff")
	die(err)
	defer expectf.Close()
	gotf, err := os.CreateTemp("", "tunk-diff")
	die(err)
	defer expectf.Close()

	_, err = expectf.Write(expect)
	die(err)
	die(expectf.Close())
	_, err = gotf.Write(got)
	die(err)
	die(gotf.Close())

	args := []string{"diff", "--no-index", "--", expectf.Name(), gotf.Name()}
	t.Logf("+ git %s", gitcli.ArgsString(args))
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err == nil {
		panic("expected error from git diff but got none")
	}
	out = bytes.ReplaceAll(out, []byte(expectf.Name()), []byte("/expect"))
	out = bytes.ReplaceAll(out, []byte(gotf.Name()), []byte("/got"))

	t.Logf("diff:\n\n%s", string(out))
}
