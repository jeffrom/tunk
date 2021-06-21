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
	flags := pflag.NewFlagSet("tunk", pflag.PanicOnError)
	flags.BoolVarP(&help, "help", "h", false, "show help")
	flags.BoolVarP(&version, "version", "V", false, "print version and exit")
	flags.BoolVarP(&cfg.Dryrun, "dry-run", "n", false, "Don't do destructive operations")
	flags.BoolVar(&cfg.All, "all", false, "operate on all scopes")
	flags.BoolVar(&cfg.Major, "major", false, "bump major version")
	flags.BoolVar(&cfg.Minor, "minor", false, "bump minor version")
	flags.BoolVar(&cfg.Patch, "patch", false, "bump patch version")
	flags.BoolVar(&cfg.InCI, "ci", false, "Run in CI mode")
	flags.BoolVar(&cfg.NoEdit, "no-edit", false, "Don't edit release tag shortlogs")
	flags.StringVarP(&cfg.Scope, "scope", "s", "", "Operate on a scope")
	flags.StringVar(&cfg.TagTemplate, "template", "", "go text/template for tag")
	flags.StringVar(&cfg.LogTemplatePath, "shortlog-template", "", "path to custom go/text template to generate shortlog")
	flags.StringArrayVarP(&cfg.Branches, "branch", "b", []string{"main", "master"}, "set release branches")
	flags.StringArrayVar(&cfg.ReleaseScopes, "release-scope", nil, "declare release scopes")
	flags.StringArrayVar(&cfg.Policies, "policy", []string{"conventional-lax", "lax"}, "declare commit policies")
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

	var rc string
	if len(args) > 0 {
		rc = args[0]
	}

	git := gitcli.New(cfg, "")
	defer git.Cleanup()
	rnr := runner.New(cfg, git)
	ctx := context.Background()
	if err := rnr.Check(ctx, rc); err != nil {
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
		tag, err := runner.RenderTag(cfg, ver)
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
