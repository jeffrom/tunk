package commit

import (
	"testing"

	"github.com/blang/semver/v4"
)

var goodVersion = semver.MustParse("1.2.3")

func TestTags(t *testing.T) {
	tcs := []struct {
		name       string
		tmpl       string
		expect     string
		expectGlob string
		semver     string
		scope      string
	}{
		{
			name:       "default",
			expect:     "v1.2.3",
			expectGlob: "v*",
		},
		{
			name:       "default-pre",
			semver:     "1.2.3-rc.0",
			expect:     "v1.2.3-rc.0",
			expectGlob: "v*",
		},
		{
			name:       "default-scope",
			expect:     "cool/v1.2.3",
			scope:      "cool",
			expectGlob: "cool/v*",
		},
		{
			name:   "no-v",
			expect: "1.2.3",
			tmpl:   `{{ .Version }}`,
		},
		{
			name:       "dash-scope",
			tmpl:       `{{- with $scope := .Version.Scope -}}{{- $scope -}}-{{- end -}}v{{- .Version -}}`,
			scope:      "cool",
			expect:     "cool-v1.2.3",
			expectGlob: "cool-v*",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tag, err := NewTag(tc.tmpl)
			if err != nil {
				t.Fatal(err)
			}

			sv := goodVersion
			if tc.semver != "" {
				sv = semver.MustParse(tc.semver)
			}

			s, err := tag.ExecuteString(TagData{Version: &Version{Version: sv, Scope: tc.scope}})
			if err != nil {
				t.Fatal(err)
			}
			t.Log("tag:", s)
			if s != tc.expect {
				t.Fatalf("expected tag %q, got %q", tc.expect, s)
			}

			if tc.expectGlob != "" {
				glob, err := tag.Glob(tc.scope, "")
				if err != nil {
					t.Fatal(err)
				}
				if glob != tc.expectGlob {
					t.Fatalf("expected glob %q, got %q", tc.expectGlob, glob)
				}
			}
		})
	}
}
