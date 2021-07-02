package commit

import (
	"github.com/blang/semver"
)

type Version struct {
	semver.Version
	Scope      string            `json:"scope,omitempty"`
	AllCommits []*AnalyzedCommit `json:"all_commits"`
	Commit     string            `json:"commit"`
	RC         string
	forGlob    bool
	forPrefix  bool
}

func (v *Version) String() string { return v.V() }

func (v *Version) V() string {
	if v.forGlob {
		if v.Version.GT(semver.Version{}) && len(v.Version.Pre) == 2 {
			globVer := v.Version
			globVer.Pre = nil
			return globVer.String()
		}
		return "*"
		// return "*.*.*"
	}
	if v.forPrefix {
		return ""
	}
	ver := v.Version
	ver.Pre = nil
	res := ver.String()
	return res
}

func (v *Version) Pre() []string {
	res := make([]string, len(v.Version.Pre))
	for i, part := range v.Version.Pre {
		res[i] = part.String()
	}
	return res
}

func (v *Version) ShortCommit() string {
	if len(v.Commit) >= 8 {
		return v.Commit[:8]
	}
	return v.Commit
}
