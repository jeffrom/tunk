package runner

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blang/semver/v4"

	"github.com/jeffrom/tunk/commit"
	"github.com/jeffrom/tunk/config"
	"github.com/jeffrom/tunk/model"
	"github.com/jeffrom/tunk/vcs/gitcli"
)

// TODO: table test that takes cfg (template, scope), version/commits,
// expect (prefix)

var defaultTestVer = &commit.Version{
	Version: semver.Version{Major: 1, Minor: 2, Patch: 3},
	AllCommits: []*commit.AnalyzedCommit{
		{
			Commit: &model.Commit{
				ID:      "deadbeef",
				Subject: "hey it's a commit",
			},
		},
		{
			Scope: "cool",
			Commit: &model.Commit{
				ID:      "deadbeef",
				Subject: "cool: hey it's a scoped commit",
			},
		},
	},
}

func TestShortlog(t *testing.T) {
	cfg := config.New(nil)
	git := gitcli.New(cfg, "")
	rnr, err := New(cfg, git)
	if err != nil {
		t.Fatal(err)
	}

	b := &bytes.Buffer{}

	ver := defaultTestVer
	if err := rnr.shortlog(context.Background(), b, ver, "test"); err != nil {
		t.Fatal(err)
	}

	res := b.String()
	// fmt.Println("log result:", res)
	expectPrefix := `test: v1.2.3

This release contains the following commits:

* hey it's a commit (deadbeef)
`

	if !strings.HasPrefix(res, expectPrefix) {
		t.Fatalf("expected prefix: %q\ngot: %q", expectPrefix, res)
	}
}

func TestShortlogScope(t *testing.T) {
	cfg := config.New(&config.Config{Scope: "cool"})
	git := gitcli.New(cfg, "")
	rnr, err := New(cfg, git)
	if err != nil {
		t.Fatal(err)
	}

	b := &bytes.Buffer{}

	ver := &*defaultTestVer
	ver.Scope = "cool"
	if err := rnr.shortlog(context.Background(), b, ver, "test"); err != nil {
		t.Fatal(err)
	}

	res := b.String()
	// fmt.Println("log result:", res)
	expectPrefix := `cool: v1.2.3

This release contains the following commits:

* cool: hey it's a scoped commit (deadbeef)
`

	if !strings.HasPrefix(res, expectPrefix) {
		t.Fatalf("expected prefix:\n\t%q\ngot:\n\t%q", expectPrefix, res)
	}
}
