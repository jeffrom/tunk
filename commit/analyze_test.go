package commit

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/blang/semver"

	"github.com/jeffrom/trunk-release/config"
	"github.com/jeffrom/trunk-release/model"
	"github.com/jeffrom/trunk-release/vcs"
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
	m := vcs.NewMock().SetTags("v0.1.0")
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

var conventionalSkipCommit = &model.Commit{ID: "deadbeef", Subject: "chore: cool chore"}
var conventionalPatchCommit = &model.Commit{ID: "deadbeef", Subject: "fix: cool fix"}
var conventionalMinorCommit = &model.Commit{ID: "deadbeef", Subject: "feat: cool feature"}
var conventionalMajorCommit = &model.Commit{ID: "deadbeef", Subject: "feat: cool feature", Body: "BREAKING CHANGE: nice breakin change"}

var conventionalScopedPatchCommit = &model.Commit{ID: "deadbeef", Subject: "fix(cool): cool fix"}

var conventionalCommits = []*model.Commit{
	commitWithID(conventionalPatchCommit, "12345678"),
	conventionalMinorCommit,
}

func commitWithID(commit *model.Commit, id string) *model.Commit {
	c := *commit
	c.ID = id
	return &c
}

// TODO from tpt-cli: goreleaser-1.0.3 is HEAD (w/ a PATCH), trunk-release is
// incorrectly searching from goreleaser-1.0.1-rc.0..HEAD because rc is
// specified ie test that we pick non-rc when there are older rcs and we're in
// rc mode
// TODO test that scoped commits aren't skipped unless allScopes len > 0

func TestAnalyzeSkip(t *testing.T) {
	tio, _, _ := mockTermIO(nil)
	cfg := config.NewWithTerminalIO(nil, &tio)
	tcs := []struct {
		name    string
		commits []*model.Commit
	}{
		{
			name:    "conventional",
			commits: []*model.Commit{conventionalSkipCommit},
		},
		{
			name:    "conventional-multi",
			commits: []*model.Commit{conventionalSkipCommit, commitWithID(conventionalSkipCommit, "12345678")},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			m := vcs.NewMock().SetTags("v0.1.0").SetCommits(tc.commits...)
			a := NewAnalyzer(cfg, m)

			vers, err := a.Analyze(context.Background(), "")
			if err != nil {
				t.Fatal(err)
			}

			if len(vers) != 0 {
				t.Fatalf("expected 0 version, got %d", len(vers))
			}
		})
	}

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
			m := vcs.NewMock().SetTags("v0.1.0").SetCommits(tc.commits...)
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
			m := vcs.NewMock().SetTags("v0.1.0").SetCommits(tc.commits...)
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

func TestAnalyzeMajor(t *testing.T) {
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
			commits:      []*model.Commit{conventionalMajorCommit},
			expectCommit: "deadbeef",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			m := vcs.NewMock().SetTags("v0.1.0").SetCommits(tc.commits...)
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
			expectVersion := semver.MustParse("1.0.0")
			if ver.Version.NE(expectVersion) {
				t.Errorf("expected version %s, got %s", expectVersion, ver.Version)
			}
		})
	}
}

func TestAnalyzeRC(t *testing.T) {
	tio, _, _ := mockTermIO(nil)

	tcs := []struct {
		name      string
		tags      []string
		scope     string
		allScopes []string
		commits   []*model.Commit
		expectTag string
	}{
		{
			name:      "patch",
			tags:      []string{"v0.1.0", "v0.1.1-rc.0"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.1",
		},
		{
			name:      "patch-two",
			tags:      []string{"v0.1.0", "v0.1.1-rc.0", "v0.1.1-rc.1"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.2",
		},
		{
			name:      "patch-missing",
			tags:      []string{"v0.1.0", "v0.1.1-rc.1"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.2",
		},
		{
			name:      "patch-nomatch",
			tags:      []string{"v0.1.0", "v0.1.1-bork.0"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.0",
		},
		{
			name:      "patch-invalid",
			tags:      []string{"v0.1.0", "v0.1.1-rc"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.0",
		},
		{
			name:      "patch-invalid",
			tags:      []string{"v0.1.0", "v0.1.1-rc.0-rc.0"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.0",
		},
		{
			name:      "patch-invalid",
			tags:      []string{"v0.1.0", "v0.1.1-rc.00"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.0",
		},
		{
			name:      "scope",
			tags:      []string{"cool-0.1.0"},
			scope:     "cool",
			allScopes: []string{"cool"},
			commits:   []*model.Commit{conventionalScopedPatchCommit},
			expectTag: "cool-0.1.1-rc.0",
		},
		{
			name:      "scope-root",
			tags:      []string{"v1.2.3", "cool-0.1.0"},
			scope:     "cool",
			allScopes: []string{"cool"},
			commits:   []*model.Commit{conventionalScopedPatchCommit},
			expectTag: "cool-0.1.1-rc.0",
		},
		{
			name:      "scope-multi",
			tags:      []string{"v1.2.3", "cool-0.1.0", "cool-0.1.1-rc.0"},
			scope:     "cool",
			allScopes: []string{"cool"},
			commits:   []*model.Commit{conventionalScopedPatchCommit},
			expectTag: "cool-0.1.1-rc.1",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.NewWithTerminalIO(&config.Config{ReleaseScopes: tc.allScopes}, &tio)
			m := vcs.NewMock().SetTags(tc.tags...).SetCommits(tc.commits...)
			a := NewAnalyzer(cfg, m)
			ver, err := a.AnalyzeScope(context.Background(), tc.scope, "rc")
			if err != nil {
				t.Fatal(err)
			}
			if ver == nil {
				t.Fatal("expected version, got none")
			}
			expectTag := tc.expectTag
			if tag := ver.GitTag(); tag != expectTag {
				t.Errorf("expected tag %q, got %q", expectTag, tag)
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
