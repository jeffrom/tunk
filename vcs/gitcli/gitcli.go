// Package gitcli implements vcs.Interface using the git commandline tool.
package gitcli

import (
	"context"

	"github.com/jeffrom/git-release/config"
	"github.com/jeffrom/git-release/vcs"
)

// Git implements vcs.Interface using the git commandline tool.
type Git struct {
	term config.TerminalIO
}

func New() *Git {
	return &Git{}
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

func (g *Git) CreateTag(ctx context.Context, commit, tag string, opts vcs.CommitOpts) error {
	return nil
}

func (g *Git) DeleteTag(ctx context.Context, commit, tag string) error {
	return nil
}

func (g *Git) QueryTags(ctx context.Context, query string) ([]string, error) {
	return nil, nil
}
