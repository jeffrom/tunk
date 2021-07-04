package commit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/blang/semver"
)

var ErrNoTags = errors.New("commit: no release tags found")

const DefaultTagTemplate = `{{- with $scope := .Version.Scope -}}
{{- $scope -}}/
{{- end -}}
v{{- semver .Version -}}`

type TagData struct {
	Version *Version
}

var funcMap = template.FuncMap{
	"join": strings.Join,
	"semver": func(v *Version) string {
		main := v.V()
		if len(v.Version.Pre) > 0 {
			return main + "-" + strings.Join(v.Pre(), ".")
		}
		return main
	},
}

type Tag struct {
	t *template.Template
}

func NewTag(s string) (*Tag, error) {
	name := ""
	if s != "" {
		name = "custom_tag"
	}
	tmpl := s
	if tmpl == "" {
		tmpl = DefaultTagTemplate
	}
	t, err := template.New(name).Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return nil, err
	}
	return &Tag{t: t}, nil
}

func (t *Tag) Execute(w io.Writer, d TagData) error {
	return t.t.Execute(w, d)
}

func (t *Tag) ExecuteString(d TagData) (string, error) {
	b := &bytes.Buffer{}
	if err := t.Execute(b, d); err != nil {
		return "", err
	}

	return b.String(), nil
}

func (t *Tag) Glob(scope, rc string) (string, error) {
	return t.ExecuteString(TagData{
		Version: &Version{forGlob: true, Scope: scope},
	})
}

func (t *Tag) GlobVersion(scope, rc string, v semver.Version) (string, error) {
	if rc != "" {
		v.Pre = []semver.PRVersion{
			{VersionStr: rc},
			{VersionStr: "*"},
		}
	}
	return t.ExecuteString(TagData{
		Version: &Version{forGlob: true, Scope: scope, Version: v},
	})
}

func (t *Tag) Prefix(scope string) (string, error) {
	return t.ExecuteString(TagData{
		Version: &Version{forPrefix: true, Scope: scope},
	})
}

func (t *Tag) ExtractSemver(scope, rc, tag string) (semver.Version, error) {
	// TODO populate TagData to render a regex that matches tags "better"?
	return extractSemver(tag)
}

func (t *Tag) SemverLatest(tags []string, scope, rc string) (semver.Version, error) {
	var versions []semver.Version
	for _, tag := range tags {
		v, err := t.ExtractSemver(scope, rc, tag)
		if err != nil {
			if errors.Is(err, errInvalidSemver) {
				continue
			}
			return semver.Version{}, err
		}

		if rc == "" && len(v.Pre) != 0 {
			continue
		} else if rc != "" && v.Pre[0].String() != rc {
			continue
		}

		if len(v.Pre) != 0 && !validTunkPre(v.Pre) {
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

var tunkPreNameRE = regexp.MustCompile(`^[A-Za-z\d]+$`)
var tunkPreNumRE = regexp.MustCompile(`^(0|[1-9]\d*)$`)

func validTunkPre(pre []semver.PRVersion) bool {
	if len(pre) != 2 {
		return false
	}
	if pre[0].IsNumeric() {
		return false
	}
	if !pre[1].IsNumeric() {
		return false
	}
	if !tunkPreNameRE.MatchString(pre[0].String()) {
		return false
	}
	if !tunkPreNumRE.MatchString(pre[1].String()) {
		return false
	}
	return true
}

// semverRE is the official semver regexp with a slight tweak (to disallow
// extra zeros at the cost of losing buildmetadata):
// https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
var semverRE = regexp.MustCompile(`(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0$|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0$|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?`)

var errInvalidSemver = errors.New("invalid semver string")

func extractSemver(s string) (semver.Version, error) {
	if !semverRE.MatchString(s) {
		return semver.Version{}, fmt.Errorf("failed to parse semver from string: %q", s)
	}
	m := semverRE.FindAllStringSubmatch(s, 1)

	v := semver.Version{}
	for _, name := range semverRE.SubexpNames() {
		if name == "" {
			continue
		}
		sub := m[0][semverRE.SubexpIndex(name)]
		if sub == "" {
			continue
		}
		isNum := true
		var subn uint64 = 0
		var err error
		subn, err = strconv.ParseUint(sub, 10, 32)
		if err != nil {
			isNum = false
		}
		switch name {
		case "major":
			v.Major = subn
		case "minor":
			v.Minor = subn
		case "patch":
			v.Patch = subn
		case "prerelease":
			if isNum {
				return semver.Version{}, errInvalidSemver
			}
			parts := strings.Split(sub, ".")
			if len(parts) != 0 && len(parts) != 2 {
				return semver.Version{}, errInvalidSemver
			}
			var pres []semver.PRVersion
			for i, part := range parts {
				switch i {
				case 0:
					if !tunkPreNameRE.MatchString(part) {
						return semver.Version{}, errInvalidSemver
					}
				case 1:
					if !tunkPreNumRE.MatchString(part) {
						return semver.Version{}, errInvalidSemver
					}
				}

				pre, err := semver.NewPRVersion(part)
				if err != nil {
					return semver.Version{}, err
				}
				pres = append(pres, pre)
			}
			v.Pre = pres
		}
		// TODO case "buildmeta"
	}

	if err := v.Validate(); err != nil {
		return semver.Version{}, err
	}
	if v.Major == 0 && v.Minor == 0 && v.Patch == 0 {
		return semver.Version{}, ErrNoTags
	}
	return v, nil
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
