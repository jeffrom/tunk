package vcs

import (
	"context"

	"github.com/jeffrom/tunk/model"
)

type Interface interface {
	Fetch(ctx context.Context, upstream, ref string) error
	Push(ctx context.Context, upstream, ref string, opts PushOpts) error
	Commit(ctx context.Context, opts CommitOpts) error
	ReadCommits(ctx context.Context, query string) ([]*model.Commit, error)
	CreateTag(ctx context.Context, commit, tag string, opts TagOpts) error
	DeleteTag(ctx context.Context, commit, tag string) error
	ReadTags(ctx context.Context, query string) ([]string, error)
	GetMainBranch(ctx context.Context, candidates []string) (string, error)
	BranchContains(ctx context.Context, commit, branch string) (bool, error)
	CurrentCommit(ctx context.Context) (string, error)
}

type CommitOpts struct {
	Message     string
	Author      string
	AuthorEmail string
}

type TagOpts struct {
	Message     string
	Commit      string
	Author      string
	AuthorEmail string
}

type PushOpts struct {
	FollowTags  bool
	GithubUser  string
	GithubToken string
}
