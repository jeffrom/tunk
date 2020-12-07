package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/pflag"

	trunkrelease "github.com/jeffrom/tunk"
	"github.com/jeffrom/tunk/commit"
	"github.com/jeffrom/tunk/config"
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
	flags.StringArrayVarP(&cfg.Branches, "branch", "b", []string{"master"}, "set release branches")
	flags.StringArrayVar(&cfg.ReleaseScopes, "scopes", nil, "declare release scopes")
	flags.StringArrayVar(&cfg.Policies, "policies", []string{"conventional-lax", "lax"}, "declare policies to use")
	flags.BoolVar(&cfg.Debug, "debug", false, "print additional debugging info")
	flags.BoolVarP(&cfg.Quiet, "quiet", "q", false, "print as little as necessary")

	if err := flags.Parse(os.Args); err != nil {
		panic(err)
	}
	args := flags.Args()[1:]

	if help {
		cfg.Printf("%s [rc]\n\nFLAGS\n%s", os.Args[0], flags.FlagUsages())
		return
	}
	if version {
		cfg.Printf("%s", trunkrelease.Version)
		return
	}

	var rc string
	if len(args) > 0 {
		rc = args[0]
	}

	a := commit.NewAnalyzer(cfg, gitcli.New(cfg, ""))
	versions, err := a.Analyze(context.Background(), rc)
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

	if cfg.Dryrun {
		return
	}
}
