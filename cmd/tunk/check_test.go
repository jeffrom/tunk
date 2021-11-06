package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type checkCommitModeTestCase struct {
	name    string
	dir     string
	ops     []testOperation
	environ []string
	gitPath string
}

func TestCheckCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("-short")
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	call(context.Background(), t, gitPath, "--version")

	// joinDir := func(paths ...string) string {
	// 	all := append([]string{"testdata", "check"}, paths...)
	// 	return Path(filepath.Join(all...))
	// }
	tcs := []checkCommitModeTestCase{
		{
			name: "basic",
			ops: []testOperation{
				{Commit: "initial commit"},
				{Tag: "v0.1.0"},
				{Commit: "feat: cool thing"},
				{TunkArgs: strs("--check")},
			},
			gitPath: gitPath,
		},
		{
			name: "fail-conventional",
			ops: []testOperation{
				{Commit: "initial commit"},
				{Tag: "v0.1.0"},
				{Commit: "cool thing"},
				{TunkArgs: strs("--check", "--policy", "conventional-lax"), ShouldFail: true},
			},
			gitPath: gitPath,
		},
		{
			name: "fail-disallowed-scope",
			ops: []testOperation{
				{Commit: "initial commit"},
				{Tag: "v0.1.0"},
				{Commit: "notnice: cool thing"},
				{TunkArgs: strs("--check", "--allowed-scope", "nice"), ShouldFail: true},
			},
			gitPath: gitPath,
		},
		{
			name: "fail-disallowed-type",
			ops: []testOperation{
				{Commit: "initial commit"},
				{Tag: "v0.1.0"},
				{Commit: "perf: cool thing"},
				{TunkArgs: strs("--check", "--allowed-type", "fix"), ShouldFail: true},
			},
			gitPath: gitPath,
		},
		{
			name: "fail-flag",
			ops: []testOperation{
				{TunkArgs: strs("--check-commit", "perf: cool", "--allowed-type", "feat"), ShouldFail: true},
			},
			gitPath: gitPath,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, runCheckCommitTest(tc))
	}
}

func runCheckCommitTest(tc checkCommitModeTestCase) func(t *testing.T) {
	return func(t *testing.T) {
		name := tc.name
		dir := tc.dir
		if dir == "" {
			dir = Path(filepath.Join("testdata", "check", tc.name))
		}
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

		tunkYAMLPath := filepath.Join(dir, "tunk.yaml")
		if _, err := os.Stat(tunkYAMLPath); err == nil {
			tunkYAML, err := os.ReadFile(tunkYAMLPath)
			die(err)
			die(os.WriteFile(filepath.Join(tmpDir, "tunk.yaml"), tunkYAML, 0644))
		}

		call(ctx, t, "git", "init")
		call(ctx, t, "git", "config", "--local", "user.email", "tunk-test@example.com")
		call(ctx, t, "git", "config", "--local", "user.name", "tunk-test")

		for _, op := range tc.ops {
			runOp(ctx, t, op)
		}
	}
}
