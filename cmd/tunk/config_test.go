package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"

	"github.com/jeffrom/tunk/config"
)

func TestInvalidConfig(t *testing.T) {
	tcs := []struct {
		name string
		args []string
	}{
		{
			name: "major-minor",
			args: strs("--major", "--minor"),
		},
		{
			name: "patch-minor",
			args: strs("--patch", "--minor"),
		},
		{
			name: "patch-major",
			args: strs("--patch", "--major"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"tunk", "--dry-run"}, tc.args...)
			t.Logf("args: %q", tc.args)
			if err := run(args); err == nil {
				t.Fatal("expected args to be invalid")
			} else {
				t.Log(err)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	defaultCfg := config.New(nil)
	tcs := []struct {
		name       string
		shouldFail bool
		dir        string
		args       []string
		expect     *config.Config
	}{
		{
			name:   "default",
			expect: &defaultCfg,
		},
	}

	dir, err := ioutil.TempDir("", "tunk-config-test")
	if err != nil {
		return
	}
	defer os.RemoveAll(dir)

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			cfgPath := filepath.Join(dir, tc.name)
			err := run(append([]string{"tunk", "--debug-config", cfgPath}, tc.args...))
			if tc.shouldFail && err == nil {
				t.Fatal("expected failure but got none")
			} else if !tc.shouldFail && err != nil {
				t.Fatal(err)
			}
			if tc.shouldFail {
				return
			}

			cfgRaw, err := ioutil.ReadFile(cfgPath)
			if err != nil {
				t.Fatal(err)
			}

			cfg := config.Config{}
			if err := yaml.Unmarshal(cfgRaw, &cfg); err != nil {
				t.Fatal(err)
			}

			expectCfg := tc.expect
			if expectCfg == nil {
				expectCfg = &config.Config{}
			}

			compareBool(t, "in_ci", cfg.InCI, expectCfg.InCI)
			compareBool(t, "debug", cfg.Debug, expectCfg.Debug)
			compareBool(t, "dryrun", cfg.Dryrun, expectCfg.Dryrun)
			compareBool(t, "quiet", cfg.Quiet, expectCfg.Quiet)
			compareBool(t, "all", cfg.All, expectCfg.All)
			compareString(t, "scope", cfg.Scope, expectCfg.Scope)
			compareString(t, "commit", cfg.Commit, expectCfg.Commit)
			compareString(t, "name", cfg.Name, expectCfg.Name)
			compareBool(t, "major", cfg.Major, expectCfg.Major)
			compareBool(t, "minor", cfg.Minor, expectCfg.Minor)
			compareBool(t, "patch", cfg.Patch, expectCfg.Patch)
			compareStrings(t, "branches", cfg.Branches, expectCfg.Branches)
			compareStrings(t, "release_scopes", cfg.ReleaseScopes, expectCfg.ReleaseScopes)
			compareStrings(t, "policies", cfg.Policies, expectCfg.Policies)
			comparePolicies(t, "custom_policies", cfg.CustomPolicies, expectCfg.CustomPolicies)
			compareString(t, "tag_template", cfg.TagTemplate, expectCfg.TagTemplate)
			compareString(t, "log_template", cfg.LogTemplate, expectCfg.LogTemplate)
			compareBool(t, "no_edit", cfg.NoEdit, expectCfg.NoEdit)
			compareStrings(t, "allowed_scopes", cfg.AllowedScopes, expectCfg.AllowedScopes)
			compareStrings(t, "allowed_types", cfg.AllowedTypes, expectCfg.AllowedTypes)
		})
	}
}

func compareBool(t testing.TB, name string, got, expect bool) {
	t.Helper()
	if got != expect {
		t.Errorf("expected %q to be %v, was %v", name, expect, got)
	}
}

func compareString(t testing.TB, name, got, expect string) {
	t.Helper()
	if got != expect {
		t.Errorf("expected %q to be %q, was %q", name, expect, got)
	}
}

func compareStrings(t testing.TB, name string, got, expect []string) {
	t.Helper()
	if len(got) != len(expect) {
		t.Errorf("expected %q to have length %d, got %d", name, len(expect), len(got))
		return
	}

	for i, cand := range got {
		expectEntry := expect[i]
		if cand != expectEntry {
			t.Errorf("expected %q#%d to be %q, was %q", name, i, cand, expectEntry)
		}
	}
}

func compareKVS(t testing.TB, name string, got, expect map[string]string) {
	t.Helper()
	if len(got) != len(expect) {
		t.Errorf("expected %q to have length %d, got %d", name, len(expect), len(got))
		return
	}

	for k, v := range got {
		expectVal, ok := expect[k]
		if !ok {
			t.Errorf("extra key %q was found in map at %q", k, name)
			continue
		}
		if v != expectVal {
			t.Errorf("expected %q key %q to be %q, was %q", name, k, expectVal, v)
		}
	}
}

func comparePolicies(t testing.TB, name string, got, expect []config.Policy) {
	t.Helper()

	if len(got) != len(expect) {
		t.Errorf("expected %q to have length %d, got %d", name, len(expect), len(got))
		return
	}

	for i, pol := range got {
		expectPol := expect[i]
		compareString(t, "name", pol.Name, expectPol.Name)
		compareString(t, "subject_regex", pol.SubjectRE, expectPol.SubjectRE)
		compareString(t, "body_annotation_start_regex", pol.BodyAnnotationStartRE, expectPol.BodyAnnotationStartRE)
		compareStrings(t, "breaking_change_types", pol.BreakingChangeTypes, expectPol.BreakingChangeTypes)
		compareKVS(t, "commit_types", pol.CommitTypes, expectPol.CommitTypes)
		compareString(t, "fallback_type", pol.FallbackReleaseType, expectPol.FallbackReleaseType)
	}
}
