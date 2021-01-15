package commit

import (
	"strings"

	"github.com/blang/semver"

	"github.com/jeffrom/tunk/model"
)

type Version struct {
	Scope      string          `json:"scope,omitempty"`
	AllCommits []*model.Commit `json:"all_commits"`
	Commit     string          `json:"commit"`
	Version    semver.Version
	RC         string
}

func (v *Version) ShortCommit() string {
	if len(v.Commit) >= 8 {
		return v.Commit[:8]
	}
	return v.Commit
}

func (v *Version) GitTag() string {
	return buildGitTag(v.Version, v.Scope, v.RC)
}

func buildGitTag(vArg semver.Version, scope, rc string) string {
	v := vArg
	// v.Pre = nil
	var b strings.Builder
	if scope != "" {
		b.WriteString(scope)
		b.WriteString("/v")
	} else {
		b.WriteString("v")
	}
	b.WriteString(v.String())

	if rc != "" {
		b.WriteString("-")
		b.WriteString(rc)
	}
	return b.String()
}
