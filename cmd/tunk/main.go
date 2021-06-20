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
		fmt.Fprintln(os.Stderr, err)
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
	flags.StringVarP(&cfg.Scope, "scope", "s", "", "Operate on a scope")
	flags.StringVar(&cfg.TagTemplate, "tag-template", "", "go text/template for tag")
	flags.StringArrayVarP(&cfg.Branches, "branch", "b", []string{"main", "master"}, "set release branches")
	flags.StringArrayVar(&cfg.ReleaseScopes, "scopes", nil, "declare release scopes")
	flags.StringArrayVar(&cfg.Policies, "policies", []string{"conventional-lax", "lax"}, "declare policies to use")
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

	var rc string
	if len(args) > 0 {
		rc = args[0]
	}

	rnr := runner.New(cfg, gitcli.New(cfg, ""))
	ctx := context.Background()
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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return nil
}

func die(err error) {
	if err != nil {
		panic(err)
	}
}

func usage(cfg config.Config, flags *pflag.FlagSet) {
	cfg.Printf("%s [rc]\n\nA utility for creating Semantic Versioning-compliant tags.\n\nFLAGS\n%s", os.Args[0], flags.FlagUsages())
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
