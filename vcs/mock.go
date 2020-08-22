package vcs

import (
	"context"
	"strings"
	"time"

	"github.com/jeffrom/git-release/model"
)

type Mock struct {
	t       time.Time
	tags    []string
	commits []*model.Commit
}

func NewMock() *Mock {
	return &Mock{
		t: time.Now(),
	}
}

func (m *Mock) SetTags(tags ...string) *Mock {
	m.tags = tags
	return m
}
func (m *Mock) SetCommits(commits ...*model.Commit) *Mock {
	finalCommits := make([]*model.Commit, len(commits))
	for i, commit := range commits {
		c := *commit
		if c.CommitterDate.IsZero() {
			c.CommitterDate = m.t
			m.t = m.t.Add(-time.Minute)
		}
		finalCommits[i] = &c
	}
	m.commits = finalCommits
	return m
}

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

func (m *Mock) ReadTags(ctx context.Context, query string) ([]string, error) {
	var tags []string
	parts := strings.Split(query, "*")
	for _, t := range m.tags {
		if hasAllParts(t, parts) {
			tags = append(tags, t)
		}
	}
	return tags, nil
}

func (m *Mock) ReadCommits(ctx context.Context, query string) ([]*model.Commit, error) {
	return m.commits, nil
}

func hasAllParts(s string, parts []string) bool {
	remaining := s
	for {
		if len(parts) == 0 {
			break
		}
		part := parts[0]
		parts = parts[1:]

		if !strings.HasPrefix(remaining, part) {
			return false
		}
		remaining = strings.TrimPrefix(remaining, part)
	}
	return true
}
