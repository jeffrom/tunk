package main

import (
	"context"
	"fmt"

	"github.com/jeffrom/git-release/commit"
	"github.com/jeffrom/git-release/config"
)

func main() {
	cfg := config.New(nil)
	a := commit.NewAnalyzer(cfg, nil)
	versions, err := a.Analyze(context.Background())
	if err != nil {
		panic(err)
	}
	for _, ver := range versions {
		fmt.Printf("version: %+v\n", ver)
	}
}
