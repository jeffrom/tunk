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
	"testing"

	"github.com/ghodss/yaml"

	"github.com/jeffrom/tunk/vcs/gitcli"
)

type testOperation struct {
	Commit   string   `json:"commit,omitempty"`
	Tag      string   `json:"tag,omitempty"`
	TunkArgs []string `json:"tunk,omitempty"`
}

func TestTunk(t *testing.T) {
	if testing.Short() {
		t.Skip("-short")
	}
	_, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	validRoot := Path("testdata/valid")
	validDirs, err := ioutil.ReadDir(validRoot)
	die(err)

	currDir, err := os.Getwd()
	die(err)

	for _, dir := range validDirs {
		name := dir.Name()
		sourceDir := filepath.Join(validRoot, name)
		t.Run(name, func(t *testing.T) {
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

			tunkYAML, err := ioutil.ReadFile(filepath.Join(sourceDir, "tunk.yaml"))
			if err == nil {
				die(ioutil.WriteFile(filepath.Join(tmpDir, "tunk.yaml"), tunkYAML, 0644))
			}
			call(ctx, t, "git", "init")
			call(ctx, t, "git", "config", "--local", "user.email", "tunk-test@example.com")
			call(ctx, t, "git", "config", "--local", "user.name", "tunk-test")

			testOpData, err := ioutil.ReadFile(filepath.Join(sourceDir, "test.yaml"))
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
				if testop.Commit != "" {
					call(ctx, t, "git", "commit", "--allow-empty", "-am", testop.Commit)
				}
				if testop.Tag != "" {
					call(ctx, t, "git", "tag", "-a", testop.Tag, "-m", testop.Tag)
				}
				if testop.TunkArgs != nil {
					callTunk(t, testop.TunkArgs...)
				}
			}
			logOut, err := exec.CommandContext(ctx,
				"git", "log", "--graph",
				"--pretty=format:%d %s",
				"--abbrev-commit",
			).Output()
			if err != nil {
				t.Fatal(err)
			}

			goldenPath := filepath.Join(sourceDir, "expect")
			if env := os.Getenv("GOLDEN"); env != "" {
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
		})
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
	cmd := exec.CommandContext(ctx, arg, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if arg == "git" {
		cmd.Env = []string{
			"GIT_AUTHOR_NAME=tunk-test",
			"GIT_AUTHOR_EMAIL=tunk-test@example.com",
			"GIT_COMMITTER_NAME=tunk-test",
			"GIT_COMMITTER_EMAIL=tunk-test@example.com",
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
