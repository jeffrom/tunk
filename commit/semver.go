package commit

import (
	"errors"
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

		v, err := semver.Parse(t)
		if err != nil {
			// cfg.Warning("invalid tag, skipping: %s", orig)
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

	semver.Sort(versions)
	if len(versions) > 0 {
		return versions[len(versions)-1], nil
	}
	return semver.Version{}, ErrNoTags
}
