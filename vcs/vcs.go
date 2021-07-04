// Package vcs abstracts version control systems. Currently just git.
package vcs

import (
	"context"
	"fmt"

	"github.com/jeffrom/tunk/model"
)

type NotFoundError struct {
	Ref string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("vcs: ref %q not found", e.Ref)
}

type Interface interface {
	Fetch(ctx context.Context, upstream, ref string) error
	Push(ctx context.Context, upstream, ref string, opts PushOpts) error
	ReadCommits(ctx context.Context, query string) ([]*model.Commit, error)
	CreateTag(ctx context.Context, commit, tag string, opts TagOpts) error
	DeleteTag(ctx context.Context, commit, tag string) error
	ReadTags(ctx context.Context, query string) ([]string, error)
	GetMainBranch(ctx context.Context, candidates []string) (string, error)
	CurrentBranch(ctx context.Context) (string, error)
	BranchContains(ctx context.Context, commit, branch string) (bool, error)
	CurrentCommit(ctx context.Context) (string, error)
	ReadNameFromRemoteURL(ctx context.Context, upstream string) (string, error)
}

type TagOpts struct {
	Message     string
	Author      string
	AuthorEmail string
}

type PushOpts struct {
	Tags       bool
	FollowTags bool
}
