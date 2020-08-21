package vcs

import "context"

type Mock struct{}

func (m *Mock) Fetch(ctx context.Context, upstream, ref string) error {
	return nil
}

func (m *Mock) Push(ctx context.Context, upstream, ref string) error {
	return nil
}

func (m *Mock) Commit(ctx context.Context, opts CommitOpts) error {
	return nil
}

func (m *Mock) CreateTag(ctx context.Context, commit, tag string, opts CommitOpts) error {
	return nil
}

func (m *Mock) DeleteTag(ctx context.Context, commit, tag string) error {
	return nil
}

func (m *Mock) QueryTags(ctx context.Context, query string) ([]string, error) {
	return nil, nil
}
