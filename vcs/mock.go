package vcs

import (
	"context"
	"strings"
	"time"

	"github.com/jeffrom/tunk/model"
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

func (m *Mock) Push(ctx context.Context, upstream, ref string, opts PushOpts) error {
	return nil
}

func (m *Mock) Commit(ctx context.Context, opts CommitOpts) error {
	return nil
}

func (m *Mock) CreateTag(ctx context.Context, commit, tag string, opts TagOpts) error {
	return nil
}

func (m *Mock) DeleteTag(ctx context.Context, commit, tag string) error {
	return nil
}

func (m *Mock) ReadTags(ctx context.Context, query string) ([]string, error) {
	var tags []string
	for _, t := range m.tags {
		// fmt.Println(query, t, globMatches(t, query))
		if globMatches(t, query) {
			tags = append(tags, t)
		}
	}
	return tags, nil
}

func (m *Mock) ReadCommits(ctx context.Context, query string) ([]*model.Commit, error) {
	return m.commits, nil
}

func globMatches(s string, glob string) bool {
	parts := strings.Split(glob, "*")
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
	if len(glob) > 0 && glob[len(glob)-1] == '*' {
		return true
	}
	return remaining == ""
}
