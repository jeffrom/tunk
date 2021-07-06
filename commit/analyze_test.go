package commit

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/blang/semver/v4"

	"github.com/jeffrom/tunk/config"
	"github.com/jeffrom/tunk/model"
	"github.com/jeffrom/tunk/vcs"
)

func TestAnalyzeNoTags(t *testing.T) {
	tio, _, _ := mockTermIO(nil)
	cfg := newTestConfig(nil, &tio)
	m := vcs.NewMock()
	a := NewAnalyzer(cfg, m, nil)

	_, err := a.Analyze(context.Background(), "")
	if err == nil {
		t.Fatal("expected no tags error")
	}
}

func TestAnalyzeNoCommits(t *testing.T) {
	tio, _, _ := mockTermIO(nil)
	cfg := newTestConfig(nil, &tio)
	m := vcs.NewMock().SetTags("v0.1.0")
	a := NewAnalyzer(cfg, m, nil)

	_, err := a.Analyze(context.Background(), "")
	if err == nil {
		t.Fatal("expected no release commits error")
	}
}

var basicPatchCommit = &model.Commit{ID: "deadbeef", Subject: "cool subject"}

// var basicCommits = []*model.Commit{
// 	basicPatchCommit,
// }

var conventionalSkipCommit = &model.Commit{ID: "deadbeef", Subject: "chore: cool chore"}
var conventionalPatchCommit = &model.Commit{ID: "deadbeef", Subject: "fix: cool fix"}
var conventionalMinorCommit = &model.Commit{ID: "deadbeef", Subject: "feat: cool feature"}
var conventionalMajorCommit = &model.Commit{ID: "deadbeef", Subject: "feat: cool feature", Body: "BREAKING CHANGE: nice breakin change"}

var conventionalScopedPatchCommit = &model.Commit{ID: "deadbeef", Subject: "fix(cool): cool fix"}

// var conventionalCommits = []*model.Commit{
// 	commitWithID(conventionalPatchCommit, "12345678"),
// 	conventionalMinorCommit,
// }

func commitWithID(commit *model.Commit, id string) *model.Commit {
	c := *commit
	c.ID = id
	return &c
}

func TestAnalyzeSkip(t *testing.T) {
	tio, _, _ := mockTermIO(nil)
	cfg := newTestConfig(&config.Config{InCI: true}, &tio)
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
			a := NewAnalyzer(cfg, m, nil)

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
	cfg := newTestConfig(nil, &tio)
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
			a := NewAnalyzer(cfg, m, nil)

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
	cfg := newTestConfig(nil, &tio)
	tcs := []struct {
		name         string
		tags         []string
		commits      []*model.Commit
		expectCommit string
	}{
		{
			name:         "conventional",
			tags:         []string{"v0.1.0"},
			commits:      []*model.Commit{conventionalMinorCommit},
			expectCommit: "deadbeef",
		},
		{
			name:         "basic+conventional",
			tags:         []string{"v0.1.0"},
			commits:      []*model.Commit{commitWithID(basicPatchCommit, "12345678"), conventionalMinorCommit},
			expectCommit: "12345678",
		},
		{
			name:         "conventional+rc",
			tags:         []string{"v0.1.0", "v0.2.0-rc.0"},
			commits:      []*model.Commit{conventionalMinorCommit},
			expectCommit: "deadbeef",
		},
		{
			name:         "conventional+rc2",
			tags:         []string{"v0.1.0", "v0.2.1-rc.0", "v0.2.0-rc.0"},
			commits:      []*model.Commit{conventionalMinorCommit},
			expectCommit: "deadbeef",
		},
		{
			name:         "conventional+rc3",
			tags:         []string{"v0.1.0", "v1.0.0-rc.0", "v0.2.0-rc.0"},
			commits:      []*model.Commit{conventionalMinorCommit},
			expectCommit: "deadbeef",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			m := vcs.NewMock().SetTags(tc.tags...).SetCommits(tc.commits...)
			a := NewAnalyzer(cfg, m, nil)

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
	cfg := newTestConfig(nil, &tio)
	tcs := []struct {
		name         string
		tags         []string
		commits      []*model.Commit
		expectCommit string
	}{
		{
			name:         "conventional",
			tags:         []string{"v0.1.0"},
			commits:      []*model.Commit{conventionalMajorCommit},
			expectCommit: "deadbeef",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			m := vcs.NewMock().SetTags(tc.tags...).SetCommits(tc.commits...)
			a := NewAnalyzer(cfg, m, nil)

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
			name:      "patch-invalid-nonum",
			tags:      []string{"v0.1.0", "v0.1.1-rc"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.0",
		},
		{
			name:      "patch-invalid-double",
			tags:      []string{"v0.1.0", "v0.1.1-rc.0-rc.0"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.0",
		},
		{
			name:      "patch-invalid-doublezero",
			tags:      []string{"v0.1.0", "v0.1.1-rc.00"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.0",
		},
		{
			name:      "patch-invalid-triplezero",
			tags:      []string{"v0.1.0", "v0.1.1-rc.000"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.0",
		},
		{
			name:      "patch-invalid-vanity-dash-prefix",
			tags:      []string{"v0.1.0", "1-v0.1.1"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.0",
		},
		{
			name:      "patch-invalid-vanity-dash-prefix",
			tags:      []string{"v0.1.0", "1/v0.1.1"},
			commits:   []*model.Commit{conventionalPatchCommit},
			expectTag: "v0.1.1-rc.0",
		},
		// TODO all of these are bugs
		// {
		// 	name:      "patch-invalid-vanity-dot",
		// 	tags:      []string{"v0.1.0", "v1.0.1.1"},
		// 	commits:   []*model.Commit{conventionalPatchCommit},
		// 	expectTag: "v0.1.1-rc.0",
		// },
		// {
		// 	name:      "patch-invalid-vanity-dash",
		// 	tags:      []string{"v0.1.0", "v1-0.1.1"},
		// 	commits:   []*model.Commit{conventionalPatchCommit},
		// 	expectTag: "v0.1.1-rc.0",
		// },
		// {
		// 	name:      "patch-invalid-vanity-dash-v-prefix",
		// 	tags:      []string{"v0.1.0", "v1-v0.1.1"},
		// 	commits:   []*model.Commit{conventionalPatchCommit},
		// 	expectTag: "v0.1.1-rc.0",
		// },
		{
			name:      "scope",
			tags:      []string{"cool/v0.1.0"},
			scope:     "cool",
			allScopes: []string{"cool"},
			commits:   []*model.Commit{conventionalScopedPatchCommit},
			expectTag: "cool/v0.1.1-rc.0",
		},
		{
			name:      "scope-root",
			tags:      []string{"v1.2.3", "cool/v0.1.0"},
			scope:     "cool",
			allScopes: []string{"cool"},
			commits:   []*model.Commit{conventionalScopedPatchCommit},
			expectTag: "cool/v0.1.1-rc.0",
		},
		{
			name:      "scope-multi",
			tags:      []string{"v1.2.3", "cool/v0.1.0", "cool/v0.1.1-rc.0"},
			scope:     "cool",
			allScopes: []string{"cool"},
			commits:   []*model.Commit{conventionalScopedPatchCommit},
			expectTag: "cool/v0.1.1-rc.1",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newTestConfig(&config.Config{ReleaseScopes: tc.allScopes}, &tio)
			m := vcs.NewMock().SetTags(tc.tags...).SetCommits(tc.commits...)
			a := NewAnalyzer(cfg, m, nil)
			ver, err := a.AnalyzeScope(context.Background(), tc.scope, "rc")
			if err != nil {
				t.Fatal(err)
			}
			if ver == nil {
				t.Fatal("expected version, got none")
			}

			tagr, err := NewTag("")
			if err != nil {
				t.Fatal(err)
			}
			tag, err := tagr.ExecuteString(TagData{Version: ver})
			if err != nil {
				t.Fatal(err)
			}

			expectTag := tc.expectTag
			if tag != expectTag {
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

func newTestConfig(overrides *config.Config, tio *config.TerminalIO) config.Config {
	cfg := config.NewWithTerminalIO(overrides, tio)
	cfg.IgnorePolicies = true
	return cfg
}
