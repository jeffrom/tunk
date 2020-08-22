package commit

import (
	"context"
	"fmt"

	"github.com/jeffrom/git-release/config"
	"github.com/jeffrom/git-release/vcs"
)

type Analyzer struct {
	vcs vcs.Interface
}

func NewAnalyzer(cfg config.Config, vcs vcs.Interface) *Analyzer {
	return &Analyzer{
		vcs: vcs,
	}
}

func (a *Analyzer) Analyze(ctx context.Context) ([]*Version, error) {
	commits, err := a.vcs.ReadCommits(ctx, "HEAD")
	if err != nil {
		return nil, err
	}
	fmt.Println(len(commits), "commits")
	return nil, nil
}

func (a *Analyzer) AnalyzeScope(ctx context.Context, scope string) ([]*Version, error) {
	return nil, nil
}

func buildTagQuery(scope string) string {
	if scope == "" {
		return "v*"
	}
	return scope + "-*"
}

func buildTagPrefix(scope string) string {
	if scope == "" {
		return "v"
	}
	return scope + "-"
}
