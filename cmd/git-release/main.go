package main

import (
	"context"
	"fmt"

	"github.com/jeffrom/git-release/commit"
	"github.com/jeffrom/git-release/config"
	"github.com/jeffrom/git-release/vcs/gitcli"
)

func main() {
	cfg := config.New(nil)
	a := commit.NewAnalyzer(cfg, gitcli.New(""))
	versions, err := a.Analyze(context.Background())
	if err != nil {
		panic(err)
	}
	for _, ver := range versions {
		fmt.Printf("version: %+v\n", ver)
	}
}
