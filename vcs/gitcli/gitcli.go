// Package gitcli implements vcs.Interface using the git commandline tool.
package gitcli

import (
	"context"
	"fmt"

	"github.com/jeffrom/git-release/config"
	"github.com/jeffrom/git-release/model"
	"github.com/jeffrom/git-release/vcs"
)

// Git implements vcs.Interface using the git commandline tool.
type Git struct {
	term config.TerminalIO
	wd   string
}

func New(wd string) *Git {
	return &Git{
		wd: wd,
	}
}

func (g *Git) Fetch(ctx context.Context, upstream, ref string) error {
	return nil
}

func (g *Git) Push(ctx context.Context, upstream, ref string) error {
	return nil
}

func (g *Git) Commit(ctx context.Context, opts vcs.CommitOpts) error {
	return nil
}

const EXPECTED_LOG_PARTS = 9

func (g *Git) ReadCommits(ctx context.Context, query string) ([]*model.Commit, error) {
	args := []string{
		"log", "--pretty=tformat:_START_%H_SEP_%aN_SEP_%ae_SEP_%ai_SEP_%cN_SEP_%ce_SEP_%ci_SEP_%s_SEP_%b_END_", query,
	}
	b, err := g.call(ctx, args)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(b))
	return nil, nil
}

func (g *Git) CreateTag(ctx context.Context, commit, tag string, opts vcs.CommitOpts) error {
	return nil
}

func (g *Git) DeleteTag(ctx context.Context, commit, tag string) error {
	return nil
}

func (g *Git) QueryTags(ctx context.Context, query string) ([]string, error) {
	return nil, nil
}
