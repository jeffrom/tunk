package commit

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/blang/semver"
	"github.com/jeffrom/git-release/config"
	"github.com/jeffrom/git-release/model"
	"github.com/jeffrom/git-release/vcs"
)

func TestAnalyzeNoTags(t *testing.T) {
	tio, _, _ := mockTermIO(nil)
	cfg := config.NewWithTerminalIO(nil, &tio)
	m := vcs.NewMock()
	a := NewAnalyzer(cfg, m)

	_, err := a.Analyze(context.Background(), "")
	if err == nil {
		t.Fatal("expected no tags error")
	}
}

func TestAnalyzeNoCommits(t *testing.T) {
	tio, _, _ := mockTermIO(nil)
	cfg := config.NewWithTerminalIO(nil, &tio)
	m := vcs.NewMock().SetTags("0.1.0")
	a := NewAnalyzer(cfg, m)

	vers, err := a.Analyze(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}

	if len(vers) != 0 {
		t.Fatalf("expected 0 versions, got %d", len(vers))
	}
}

var basicPatchCommit = &model.Commit{ID: "deadbeef", Subject: "cool subject"}

var basicCommits = []*model.Commit{
	basicPatchCommit,
}

var conventionalPatchCommit = &model.Commit{ID: "deadbeef", Subject: "fix: cool fix"}
var conventionalMinorCommit = &model.Commit{ID: "deadbeef", Subject: "feat: cool feature"}

var conventionalCommits = []*model.Commit{
	commitWithID(conventionalPatchCommit, "12345678"),
	conventionalMinorCommit,
}

func commitWithID(commit *model.Commit, id string) *model.Commit {
	c := *commit
	c.ID = id
	return &c
}

func TestAnalyzePatch(t *testing.T) {
	tio, _, _ := mockTermIO(nil)
	cfg := config.NewWithTerminalIO(nil, &tio)
	tcs := []struct {
		name         string
		tags         []string
		commits      []*model.Commit
		expectCommit string
	}{
		{
			name:         "basic",
			commits:      []*model.Commit{basicPatchCommit},
			expectCommit: "deadbeef",
		},
		{
			name:         "conventional",
			commits:      []*model.Commit{conventionalPatchCommit},
			expectCommit: "deadbeef",
		},
		{
			name:         "basic+conventional",
			commits:      []*model.Commit{commitWithID(basicPatchCommit, "12345678"), conventionalPatchCommit},
			expectCommit: "12345678",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			m := vcs.NewMock().SetTags("0.1.0").SetCommits(tc.commits...)
			a := NewAnalyzer(cfg, m)

			vers, err := a.Analyze(context.Background(), "")
			if err != nil {
				t.Fatal(err)
			}

			if len(vers) != 1 {
				t.Fatalf("expected 1 version, got %d", len(vers))
			}
			ver := vers[0]
			if expectCommit := tc.expectCommit; expectCommit != "" {
				if ver.Commit != expectCommit {
					t.Errorf("expected commit %q, got %q", expectCommit, ver.Commit)
				}
			}
			expectVersion := semver.MustParse("0.1.1")
			if ver.Version.NE(expectVersion) {
				t.Errorf("expected version %s, got %s", expectVersion, ver.Version)
			}
		})
	}
}

func TestAnalyzeMinor(t *testing.T) {
	tio, _, _ := mockTermIO(nil)
	cfg := config.NewWithTerminalIO(nil, &tio)
	tcs := []struct {
		name         string
		tags         []string
		commits      []*model.Commit
		expectCommit string
	}{
		{
			name:         "conventional",
			commits:      []*model.Commit{conventionalMinorCommit},
			expectCommit: "deadbeef",
		},
		{
			name:         "basic+conventional",
			commits:      []*model.Commit{commitWithID(basicPatchCommit, "12345678"), conventionalMinorCommit},
			expectCommit: "12345678",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			m := vcs.NewMock().SetTags("0.1.0").SetCommits(tc.commits...)
			a := NewAnalyzer(cfg, m)

			vers, err := a.Analyze(context.Background(), "")
			if err != nil {
				t.Fatal(err)
			}

			if len(vers) != 1 {
				t.Fatalf("expected 1 version, got %d", len(vers))
			}
			ver := vers[0]
			if expectCommit := tc.expectCommit; expectCommit != "" {
				if ver.Commit != expectCommit {
					t.Errorf("expected commit %q, got %q", expectCommit, ver.Commit)
				}
			}
			expectVersion := semver.MustParse("0.2.0")
			if ver.Version.NE(expectVersion) {
				t.Errorf("expected version %s, got %s", expectVersion, ver.Version)
			}
		})
	}
}

func mockTermIO(stdin io.Reader) (config.TerminalIO, *bytes.Buffer, *bytes.Buffer) {
	ob := &bytes.Buffer{}
	eb := &bytes.Buffer{}
	tio := config.TerminalIO{Stdin: stdin, Stdout: ob, Stderr: eb}
	return tio, ob, eb
}
