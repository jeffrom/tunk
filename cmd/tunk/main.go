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
	"github.com/jeffrom/tunk/runner"
	"github.com/jeffrom/tunk/vcs/gitcli"
)

var (
	// these are overridden by go build -X
	ShareDir string
	Version  string
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
	var debugConfig string
	var printConfig bool
	var printLatest bool
	flags := pflag.NewFlagSet("tunk", pflag.ExitOnError)
	flags.BoolVarP(&help, "help", "h", false, "show help")
	flags.BoolVarP(&version, "version", "V", false, "print version and exit")
	flags.BoolVarP(&cfg.Dryrun, "dry-run", "n", false, "Don't do destructive operations")
	flags.BoolVarP(&cfg.All, "all", "a", false, "operate on all scopes")
	flags.BoolVar(&cfg.Major, "major", false, "bump major version")
	flags.BoolVar(&cfg.Minor, "minor", false, "bump minor version")
	flags.BoolVar(&cfg.Patch, "patch", false, "bump patch version")
	flags.BoolVar(&cfg.InCI, "ci", false, "Run in CI mode")
	flags.BoolVarP(&readStats, "stats", "S", false, "print repository stats (with top tens)")
	flags.BoolVarP(&readAllStats, "stats-all", "A", false, "print all repository stats")
	flags.BoolVarP(&cfg.NoEdit, "no-edit", "E", false, "Don't edit release tag shortlogs")
	flags.StringVarP(&cfg.Scope, "scope", "s", "", "Operate on the `name`d scope")
	flags.StringVar(&cfg.TagTemplate, "template", "", "go text/template for tag `format`")
	flags.StringVar(&cfg.LogTemplate, "shortlog-template", "", "path to custom shortlog go/text template `format`")
	flags.StringArrayVarP(&cfg.Branches, "branch", "b", []string{"main", "master"}, "set release branch to `name`")
	flags.StringArrayVar(&cfg.ReleaseScopes, "release-scope", nil, "declare release scopes' `name`s")
	flags.StringArrayVar(&cfg.AllowedScopes, "allowed-scope", nil, "declare allowed scopes' `name`s")
	flags.StringArrayVar(&cfg.AllowedTypes, "allowed-type", nil, "declare allowed commit `type`s")
	flags.StringArrayVar(&cfg.Policies, "policy", []string{"conventional-lax", "lax"}, "declare commit policies by `name`")
	flags.BoolVarP(&noPolicy, "no-policy", "P", false, "disable all commit policies")
	flags.StringArrayVar(&checkCommits, "check-commit", nil, "only validate provided commit `body`")
	flags.BoolVarP(&checkCommitsFromGit, "check", "C", false, "only validate commits since last release")
	flags.StringVar(&cfg.Name, "name", "", "name the project")
	flags.BoolVarP(&cfg.Debug, "verbose", "v", false, "print additional debugging info")
	flags.BoolVarP(&cfg.Quiet, "quiet", "q", false, "print as little as necessary")
	flags.StringVarP(&cfgFile, "config", "c", "", "specify config `file`")
	flags.BoolVar(&printConfig, "print-config", false, "Print default configuration and exit")
	flags.BoolVar(&printLatest, "latest", false, "Print latest version and exit")
	flags.StringVar(&debugConfig, "debug-config", "", "Write configuration to `file` and exit")

	if err := flags.Parse(rawArgs); err != nil {
		return err
	}
	args := flags.Args()[1:]

	if help {
		usage(cfg, flags)
		return nil
	}
	if version {
		cfg.Printf("%s", Version)
		return nil
	}
	if printConfig {
		b, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		cfg.Printf("%s", string(b))
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
		if err := mergo.Merge(&cfg, tunkYAML, mergo.WithOverride); err != nil {
			return err
		}

		if tunkYAML.Branches != nil && len(tunkYAML.Branches) == 0 && !flags.Lookup("branch").Changed {
			cfg.Branches = tunkYAML.Branches
		}
		if tunkYAML.Policies != nil && len(tunkYAML.Policies) == 0 && !flags.Lookup("policy").Changed {
			cfg.Policies = tunkYAML.Policies
		}
	}
	if cfg.Debug {
		b, err := json.MarshalIndent(cfg, "", "  ")
		die(err)
		cfg.Debugf("config: %s", string(b))
	}
	branchesSet := false
	if fl := flags.Lookup("branch"); fl != nil && fl.Changed {
		branchesSet = true
	}
	if tunkYAML != nil && tunkYAML.Branches != nil {
		branchesSet = true
	}
	cfg.BranchesSet = branchesSet
	if noPolicy {
		cfg.Policies = nil
	}

	if debugConfig != "" {
		b, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		if debugConfig == "-" {
			cfg.Printf("%s", b)
		} else {
			if err := ioutil.WriteFile(debugConfig, b, 0644); err != nil {
				return err
			}
		}
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	if debugConfig != "" {
		return nil
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

	stdoutfd := os.Stdout.Fd()
	istty := isatty.IsTerminal(stdoutfd)

	if printLatest {
		latest, err := rnr.LatestRelease(ctx, cfg.Scope, rc)
		if err != nil {
			return err
		}
		tagTmpl, err := commit.NewTag(cfg.TagTemplate)
		if err != nil {
			return err
		}
		tag, err := runner.RenderTag(cfg, tagTmpl, &commit.Version{Version: latest})
		if err != nil {
			return err
		}
		if cfg.Quiet || !istty {
			fmt.Fprintf(cfg.Term.Stdout, "%s", tag)
		} else {
			fmt.Fprintln(cfg.Term.Stdout, tag)
		}
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

	for _, ver := range versions {
		tag, err := runner.RenderTag(cfg, tag, ver)
		if err != nil {
			return err
		}
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

See the following man pages for more information:

man tunk
man 5 tunk-config
man tunk-ci

EXAMPLES

# bump the version, if there are any new commits
$ tunk

# bump the minor version regardless of the state of the branch.
$ tunk --minor

# bump the version for scope "myscope" only
$ tunk -s myscope

# bump the version for all release scopes (can be defined in tunk.yaml)
$ tunk --all --release-scope myscope --release-scope another-scope

# validate against policies, allowed scopes, and allowed types:
$ tunk --check
`, os.Args[0], flags.FlagUsages())
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
