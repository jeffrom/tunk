package main

import (
	"os"
	"path/filepath"
	"strings"
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

type Conf = config.Config

func TestLoadConfig(t *testing.T) {
	defaultCfg := config.New(nil)
	newCfg := func(c *config.Config) *config.Config {
		cfg := config.New(c)
		return &cfg
	}
	editDefaultCfg := func(fn func(c config.Config) config.Config) *config.Config {
		cfg := config.New(nil)
		cfg = fn(cfg)
		return &cfg
	}
	tcs := []struct {
		name       string
		shouldFail bool
		tunkYAML   string
		args       []string
		environ    []string
		expect     *config.Config
	}{
		{
			name:   "default",
			expect: &defaultCfg,
		},
		{
			name:     "release-scopes",
			tunkYAML: `release_scopes: [nice, cool]`,
			expect:   newCfg(&Conf{ReleaseScopes: strs("nice", "cool")}),
		},
		{
			name:     "branches",
			tunkYAML: `branches: [master]`,
			expect:   newCfg(&Conf{Branches: strs("master")}),
		},
		{
			name:     "branches-unset",
			tunkYAML: `branches: []`,
			expect: editDefaultCfg(func(c Conf) Conf {
				c.Branches = nil
				return c
			}),
		},
		{
			name:     "branches-unset-override",
			tunkYAML: `branches: []`,
			args:     strs("--branch", "bork"),
			expect:   newCfg(&Conf{Branches: strs("bork")}),
		},
		{
			name:     "policies",
			tunkYAML: `policies: [conventional-lax]`,
			expect:   newCfg(&Conf{Policies: strs("conventional-lax")}),
		},
		{
			name:     "policies-unset",
			tunkYAML: `policies: []`,
			expect: editDefaultCfg(func(c Conf) Conf {
				c.Policies = nil
				return c
			}),
		},
		{
			name:     "policies-unset-override",
			tunkYAML: `policies: []`,
			args:     strs("--policy", "conventional-lax"),
			expect:   newCfg(&Conf{Policies: strs("conventional-lax")}),
		},
	}

	dir, err := os.MkdirTemp("", "tunk-config-test")
	if err != nil {
		return
	}
	defer os.RemoveAll(dir)

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			currEnv := os.Environ()
			defer resetEnviron(t, currEnv)
			os.Clearenv()
			for _, env := range tc.environ {
				parts := strings.SplitN(env, "=", 2)
				key, val := parts[0], parts[1]
				die(os.Setenv(key, val))
			}

			cleanTestName := strings.ReplaceAll(t.Name(), string(filepath.Separator), "-")
			tunkYAMLSourcePath := filepath.Join(dir, cleanTestName+".tunk.yaml")
			cfgPath := filepath.Join(dir, cleanTestName)

			testArgs := []string{"tunk", "--debug-config", cfgPath}
			if tc.tunkYAML != "" {
				if err := os.WriteFile(tunkYAMLSourcePath, []byte(tc.tunkYAML), 0644); err != nil {
					t.Fatal(err)
				}
				testArgs = append(testArgs, "-c", tunkYAMLSourcePath)
			}

			err := run(append(testArgs, tc.args...))
			if tc.shouldFail && err == nil {
				t.Fatal("expected failure but got none")
			} else if !tc.shouldFail && err != nil {
				t.Fatal(err)
			}
			if tc.shouldFail {
				return
			}

			cfgRaw, err := os.ReadFile(cfgPath)
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
		compareStrings(t, "breaking_change_annotations", pol.BreakingChangeTypes, expectPol.BreakingChangeTypes)
		compareKVS(t, "commit_types", pol.CommitTypes, expectPol.CommitTypes)
		compareString(t, "fallback_type", pol.FallbackReleaseType, expectPol.FallbackReleaseType)
	}
}
