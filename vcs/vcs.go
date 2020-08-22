package vcs

import (
	"context"

	"github.com/jeffrom/git-release/model"
)

type Interface interface {
	Fetch(ctx context.Context, upstream, ref string) error
	Push(ctx context.Context, upstream, ref string) error
	Commit(ctx context.Context, opts CommitOpts) error
	ReadCommits(ctx context.Context, query string) ([]*model.Commit, error)
	CreateTag(ctx context.Context, commit, tag string, opts CommitOpts) error
	DeleteTag(ctx context.Context, commit, tag string) error
	QueryTags(ctx context.Context, query string) ([]string, error)
}

type CommitOpts struct {
	Message string
}
