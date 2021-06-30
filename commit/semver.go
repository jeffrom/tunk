package commit

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/blang/semver"
)

var ErrNoTags = errors.New("commit: no release tags found")

func semverLatest(tags []string, scope, prerelease string) (semver.Version, error) {
	var versions []semver.Version
	for _, t := range tags {
		// orig := t
		if strings.HasPrefix(t, "v") {
			t = t[1:]
		} else if scope != "" && strings.HasPrefix(t, scope+"/") {
			parts := strings.SplitN(t, "/", 2)
			t = parts[1]
		}

		v, err := semver.ParseTolerant(t)
		if err != nil {
			fmt.Printf("invalid tag, skipping: %s\n", t)
			continue
		}

		// skip release candidates if they don't match. v.Pre splits on
		// periods, so -rc.0 will be [rc, 0]
		if prerelease == "" && len(v.Pre) > 0 {
			continue
		} else if prerelease != "" && (len(v.Pre) != 2 || v.Pre[0].String() != prerelease) {
			continue
		}

		versions = append(versions, v)
	}

	sort.Sort(tunkVersions(versions))
	// fmt.Println("sorted tags:", versions)
	if len(versions) > 0 {
		return versions[len(versions)-1], nil
	}
	return semver.Version{}, ErrNoTags
}

type tunkVersions []semver.Version

func (s tunkVersions) Len() int { return len(s) }
func (s tunkVersions) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less implements sort.Interface. It takes into account tunks rc tag structure
// (1.2.3-myrc.N)
func (s tunkVersions) Less(i, j int) bool {
	a, b := s[i], s[j]
	if a.Major != b.Major || a.Minor != b.Minor || a.Patch != b.Patch {
		return a.LT(b)
	}

	if len(a.Pre) == 2 && len(b.Pre) == 2 {
		if a.Pre[0] != b.Pre[0] {
			return a.LT(b)
		}

		if !a.Pre[1].IsNum || !b.Pre[1].IsNum {
			return a.LT(b)
		}

		return a.Pre[1].VersionNum < b.Pre[1].VersionNum
	}
	return a.LT(b)
}
