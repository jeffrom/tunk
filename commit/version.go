package commit

import (
	"strings"

	"github.com/blang/semver"

	"github.com/jeffrom/git-release/model"
)

type Version struct {
	Scope      string          `json:"scope,omitempty"`
	AllCommits []*model.Commit `json:"all_commits"`
	Commit     string          `json:"commit"`
	Version    semver.Version
	RC         string
}

func (v *Version) GitTag() string {
	var b strings.Builder
	if v.Scope != "" {
		b.WriteString(v.Scope)
		b.WriteString("-")
	} else {
		b.WriteString("v")
	}
	b.WriteString(v.Version.String())

	if v.RC != "" {
		b.WriteString("-")
		b.WriteString(v.RC)
	}
	return b.String()
}
