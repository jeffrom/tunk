package main

import (
	"context"
	"os"

	"github.com/spf13/pflag"

	trunkrelease "github.com/jeffrom/trunk-release"
	"github.com/jeffrom/trunk-release/commit"
	"github.com/jeffrom/trunk-release/config"
	"github.com/jeffrom/trunk-release/vcs/gitcli"
)

func main() {
	cfg := config.New(nil)

	var help bool
	var version bool
	flags := pflag.NewFlagSet("trunk-release", pflag.PanicOnError)
	flags.BoolVarP(&help, "help", "h", false, "show help")
	flags.BoolVarP(&version, "version", "V", false, "print version and exit")
	flags.BoolVar(&cfg.Force, "force", false, "force destructive operations")
	flags.BoolVarP(&cfg.Dryrun, "dry-run", "n", false, "Don't do destructive operations")
	flags.BoolVar(&cfg.All, "all", false, "operate on all scopes")
	flags.StringVarP(&cfg.Scope, "scope", "s", "", "Operate on a scope")
	flags.StringArrayVar(&cfg.ReleaseScopes, "scopes", nil, "declare release scopes")
	flags.StringArrayVar(&cfg.Policies, "policies", []string{"conventional-lax", "lax"}, "declare policies to use")
	flags.BoolVar(&cfg.Debug, "debug", false, "print additional debugging info")
	flags.BoolVarP(&cfg.Quiet, "quiet", "q", false, "only print errors")

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
		panic(err)
	}
	cfg.Printf("will tag %d:", len(versions))
	for _, ver := range versions {
		cfg.Printf("-> %s:%s", ver.ShortCommit(), ver.GitTag())
	}
}
