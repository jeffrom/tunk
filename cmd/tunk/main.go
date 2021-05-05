package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"

	"github.com/jeffrom/tunk/config"
	"github.com/jeffrom/tunk/release"
	"github.com/jeffrom/tunk/runner"
	"github.com/jeffrom/tunk/vcs/gitcli"
)

func main() {
	cfg := config.New(nil)

	var help bool
	var version bool
	flags := pflag.NewFlagSet("tunk", pflag.PanicOnError)
	flags.BoolVarP(&help, "help", "h", false, "show help")
	flags.BoolVarP(&version, "version", "V", false, "print version and exit")
	flags.BoolVarP(&cfg.Dryrun, "dry-run", "n", false, "Don't do destructive operations")
	flags.BoolVar(&cfg.All, "all", false, "operate on all scopes")
	flags.StringVarP(&cfg.Scope, "scope", "s", "", "Operate on a scope")
	flags.StringArrayVarP(&cfg.Branches, "branch", "b", []string{"main", "master"}, "set release branches")
	flags.StringArrayVar(&cfg.ReleaseScopes, "scopes", nil, "declare release scopes")
	flags.StringArrayVar(&cfg.Policies, "policies", []string{"conventional-lax", "lax"}, "declare policies to use")
	flags.BoolVarP(&cfg.Debug, "verbose", "v", false, "print additional debugging info")
	flags.BoolVarP(&cfg.Quiet, "quiet", "q", false, "print as little as necessary")

	if err := flags.Parse(os.Args); err != nil {
		panic(err)
	}
	args := flags.Args()[1:]

	if help {
		usage(cfg, flags)
		return
	}
	if version {
		cfg.Printf("%s", release.Version)
		return
	}

	var rc string
	if len(args) > 0 {
		rc = args[0]
	}

	runner := runner.New(cfg, gitcli.New(cfg, ""))
	ctx := context.Background()
	versions, err := runner.Analyze(ctx, rc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cfg.Debugf("will tag %d:", len(versions))

	stdoutfd := os.Stdout.Fd()
	istty := isatty.IsTerminal(stdoutfd)
	for _, ver := range versions {
		if cfg.Quiet {
			tag := ver.GitTag()
			if istty {
				fmt.Println(tag)
			} else {
				fmt.Print(tag)
			}
		} else {
			cfg.Printf("-> %s:%s", ver.ShortCommit(), ver.GitTag())
		}
	}

	if err := runner.CreateTags(ctx, versions); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage(cfg config.Config, flags *pflag.FlagSet) {
	cfg.Printf("%s [rc]\n\nA utility for creating Semantic Versioning-compliant tags.\n\nFLAGS\n%s", os.Args[0], flags.FlagUsages())
}
