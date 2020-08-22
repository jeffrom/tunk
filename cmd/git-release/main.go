package main

import (
	"context"
	"os"

	"github.com/spf13/pflag"

	gitrelease "github.com/jeffrom/git-release"
	"github.com/jeffrom/git-release/commit"
	"github.com/jeffrom/git-release/config"
	"github.com/jeffrom/git-release/vcs/gitcli"
)

func main() {
	cfg := config.New(nil)

	var help bool
	var version bool
	flags := pflag.NewFlagSet("git-release", pflag.PanicOnError)
	flags.BoolVarP(&help, "help", "h", false, "show help")
	flags.BoolVarP(&version, "version", "V", false, "print version and exit")
	flags.BoolVar(&cfg.Force, "force", false, "force destructive operations")
	flags.BoolVarP(&cfg.Dryrun, "dry-run", "n", false, "Don't do destructive operations")
	flags.StringVarP(&cfg.Scope, "scope", "s", "", "Only run on a single scope")
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
		cfg.Printf("%s", gitrelease.Version)
		return
	}

	var rc string
	if len(args) > 0 {
		rc = args[0]
	}

	a := commit.NewAnalyzer(cfg, gitcli.New(""))
	versions, err := a.Analyze(context.Background(), rc)
	if err != nil {
		panic(err)
	}
	for _, ver := range versions {
		cfg.Debugf("will tag: %s:%s", ver.ShortCommit(), ver.GitTag())
	}
}
