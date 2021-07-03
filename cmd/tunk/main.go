package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"

	"github.com/jeffrom/tunk/commit"
	"github.com/jeffrom/tunk/config"
	"github.com/jeffrom/tunk/release"
	"github.com/jeffrom/tunk/runner"
	"github.com/jeffrom/tunk/vcs/gitcli"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(rawArgs []string) error {
	cfg := config.New(nil)

	var help bool
	var version bool
	var cfgFile string
	var noPolicy bool
	var checkCommits []string
	var checkCommitsFromGit bool
	var readStats bool
	var readAllStats bool
	flags := pflag.NewFlagSet("tunk", pflag.PanicOnError)
	flags.BoolVarP(&help, "help", "h", false, "show help")
	flags.BoolVarP(&version, "version", "V", false, "print version and exit")
	flags.BoolVarP(&cfg.Dryrun, "dry-run", "n", false, "Don't do destructive operations")
	flags.BoolVar(&cfg.All, "all", false, "operate on all scopes")
	flags.BoolVar(&cfg.Major, "major", false, "bump major version")
	flags.BoolVar(&cfg.Minor, "minor", false, "bump minor version")
	flags.BoolVar(&cfg.Patch, "patch", false, "bump patch version")
	flags.BoolVar(&cfg.InCI, "ci", false, "Run in CI mode")
	flags.BoolVarP(&readStats, "stats", "S", false, "print repository stats (with top tens)")
	flags.BoolVarP(&readAllStats, "stats-all", "A", false, "print all repository stats")
	flags.BoolVarP(&cfg.NoEdit, "no-edit", "E", false, "Don't edit release tag shortlogs")
	flags.StringVarP(&cfg.Scope, "scope", "s", "", "Operate on a scope")
	flags.StringVar(&cfg.TagTemplate, "template", "", "go text/template for tag format")
	flags.StringVar(&cfg.LogTemplatePath, "shortlog-template", "", "path to custom go/text template to generate shortlog")
	flags.StringArrayVarP(&cfg.Branches, "branch", "b", []string{"main", "master"}, "set release branches")
	flags.StringArrayVar(&cfg.ReleaseScopes, "release-scope", nil, "declare release scopes")
	flags.StringArrayVar(&cfg.AllowedScopes, "allowed-scope", nil, "declare allowed scopes")
	flags.StringArrayVar(&cfg.AllowedTypes, "allowed-type", nil, "declare allowed commit types")
	flags.StringArrayVar(&cfg.Policies, "policy", []string{"conventional-lax", "lax"}, "declare commit policies")
	flags.BoolVarP(&noPolicy, "no-policy", "P", false, "disable all commit policies")
	flags.StringArrayVarP(&checkCommits, "check-commit", "C", nil, "only validate commits")
	flags.BoolVar(&checkCommitsFromGit, "check", false, "only validate commits since last release")
	flags.BoolVarP(&cfg.Debug, "verbose", "v", false, "print additional debugging info")
	flags.BoolVarP(&cfg.Quiet, "quiet", "q", false, "print as little as necessary")
	flags.StringVarP(&cfgFile, "config", "c", "", "specify config file")

	die(flags.Parse(rawArgs))
	args := flags.Args()[1:]

	if help {
		usage(cfg, flags)
		return nil
	}
	if version {
		cfg.Printf("%s", release.Version)
		return nil
	}
	if !cfg.InCI {
		if env := os.Getenv("CI"); env == "true" || env == "1" || env == "yes" {
			cfg.InCI = true
		}
	}

	tunkYAML, err := readTunkYAML(cfgFile)
	if err != nil {
		return err
	}
	if tunkYAML != nil {
		die(mergo.Merge(&cfg, tunkYAML, mergo.WithOverride))
	}
	if cfg.Debug {
		b, err := json.MarshalIndent(cfg, "", "  ")
		die(err)
		cfg.Debugf("config: %s", string(b))
	}
	branchesSet := false
	if fl := flags.Lookup("branch"); fl != nil {
		if fl.Changed {
			branchesSet = true
		}
	}
	if tunkYAML != nil {
		if len(tunkYAML.Branches) > 0 {
			branchesSet = true
		}
	}
	cfg.BranchesSet = branchesSet
	if noPolicy {
		cfg.Policies = nil
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	// done setting up config

	var rc string
	if len(args) > 0 {
		rc = args[0]
	}

	git := gitcli.New(cfg, "")
	defer git.Cleanup()
	rnr, err := runner.New(cfg, git)
	if err != nil {
		return err
	}
	ctx := context.Background()

	if readStats || readAllStats {
		stats, err := rnr.Stats(ctx)
		if err != nil {
			return err
		}
		if err := stats.TextSummary(cfg.Term.Stdout, readAllStats); err != nil {
			return err
		}
		return nil
	}

	shouldCheckCommits := checkCommitsFromGit || flags.Lookup("check-commit").Changed
	if shouldCheckCommits {
		hasPipe := !isatty.IsTerminal(os.Stdin.Fd())
		var err error
		if checkCommitsFromGit {
			err = rnr.CheckCommitsFromGit(ctx, cfg.Scope)
		} else if hasPipe && len(checkCommits) == 1 && checkCommits[0] == "-" {
			err = rnr.CheckReadCommits(ctx, os.Stdin)
		} else {
			err = rnr.CheckCommitSubjects(ctx, checkCommits)
		}
		if err != nil {
			cf := runner.CheckFailure{}
			if errors.As(err, &cf) {
				if err := cf.WriteFailure(os.Stdout); err != nil {
					fmt.Fprintln(os.Stderr, "failed to write invalid commit information:", err)
				}
			}
			return err
		}
		cfg.Printf("OK")
		return nil
	}

	if err := rnr.Check(ctx, rc); err != nil {
		return err
	}

	tag, err := commit.NewTag(cfg.TagTemplate)
	if err != nil {
		return err
	}

	versions, err := rnr.Analyze(ctx, rc)
	if err != nil {
		return err
	}
	cfg.Debugf("will tag %d:", len(versions))

	stdoutfd := os.Stdout.Fd()
	istty := isatty.IsTerminal(stdoutfd)
	for _, ver := range versions {
		tag, err := runner.RenderTag(cfg, tag, ver)
		die(err)
		if cfg.Quiet {
			if istty {
				fmt.Println(tag)
			} else {
				fmt.Print(tag)
			}
		} else {
			cfg.Printf("-> %s:%s", ver.ShortCommit(), tag)
		}
	}

	if err := rnr.CreateTags(ctx, versions); err != nil {
		return err
	}

	if cfg.InCI && len(versions) > 0 {
		cfg.Printf("Pushing tags in CI mode...")
		if err := rnr.PushTags(ctx); err != nil {
			return err
		}
	}
	return nil
}

func die(err error) {
	if err != nil {
		panic(err)
	}
}

func usage(cfg config.Config, flags *pflag.FlagSet) {
	cfg.Printf(`%s [rc]

A utility for creating Semantic Version-compliant tags.

FLAGS
%s
EXAMPLES

# bump the version, if there are any new commits
$ tunk

# bump the minor version regardless of the state of the branch.
$ tunk --minor

# bump the version for scope "myscope" only
$ tunk -s myscope

# bump the version for all release scopes (can be defined in tunk.yaml)
$ tunk --all --release-scope myscope --release-scope another-scope

TEMPLATING

Tags can be read and written according to a template. The only requirement is
that a SemVer-compliant version is included as one continuous string, including
the prerelease portion. See "go doc github.com/jeffrom/tunk/commit TagData" for
more information.

The default tag template is:

%s

A template that doesn't include the "v" prefix, and changes the scope
delineator from "/" to "#" could look like this:

{{- with $scope := .Version.Scope -}}
{{- $scope -}}#
{{- end -}}
{{- .Version -}}
{{- with $pre := .Version.Pre -}}
-{{- join $pre "." -}}
{{- end -}}

VALIDATION

tunk can validate against policies, allowed scopes, and allowed types:

$ tunk --check
`, os.Args[0], flags.FlagUsages(), commit.DefaultTagTemplate)
}

func readTunkYAML(p string) (*config.Config, error) {
	if p != "" {
		b, err := ioutil.ReadFile(p)
		if err != nil {
			return nil, err
		}
		cfg := &config.Config{}
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for {
		candPath := filepath.Join(wd, "tunk.yaml")
		b, err := ioutil.ReadFile(candPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				wd, _ = filepath.Split(filepath.Clean(wd))
				if wd == "/" {
					break
				}
				continue
			}
			return nil, err
		}

		cfg := &config.Config{}
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}
	return nil, nil
}
