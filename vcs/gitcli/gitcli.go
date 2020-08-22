// Package gitcli implements vcs.Interface using the git commandline tool.
package gitcli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

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

	var commits []*model.Commit
	scanner := bufio.NewScanner(bytes.NewBuffer(b))
	for scanner.Scan() {
		s := scanner.Text()
		parts := strings.Split(s, "_SEP_")
		if len(parts) != EXPECTED_LOG_PARTS {
			return nil, fmt.Errorf("gitcli: expected %d parts from git log, got %d", EXPECTED_LOG_PARTS, len(parts))
		}

		commitID := parts[0]
		if !strings.HasPrefix(commitID, "_START_") {
			return nil, fmt.Errorf("gitcli: unexpected git log line: %q", s)
		}
		commitID = strings.TrimPrefix(commitID, "_START_")

		// body can be multiple lines.
		var body string
		bodypart := parts[len(parts)-1]
		if strings.HasSuffix(bodypart, "_END_") {
			body = strings.TrimSuffix(bodypart, "_END_")
		} else {
			var bodyb strings.Builder
			bodyb.WriteString(bodypart)
			bodyb.WriteString("\n")
			for scanner.Scan() {
				bodyline := scanner.Text()
				if strings.HasSuffix(bodyline, "_END_") {
					if trimmed := strings.TrimSpace(strings.TrimSuffix(bodyline, "_END_")); trimmed != "" {
						bodyb.WriteString(trimmed)
					}
					break
				}
				bodyb.WriteString(bodyline)
				bodyb.WriteString("\n")
			}
			body = bodyb.String()
		}

		authorDateStr := parts[3]
		authorDate, err := ParseGitISO8601(authorDateStr)
		if err != nil {
			return nil, err
		}
		committerDateStr := parts[6]
		committerDate, err := ParseGitISO8601(committerDateStr)
		if err != nil {
			return nil, err
		}

		commits = append(commits, &model.Commit{
			ID:             commitID,
			Author:         parts[1],
			AuthorEmail:    parts[2],
			AuthorDate:     authorDate,
			Committer:      parts[4],
			CommitterEmail: parts[5],
			CommitterDate:  committerDate,
			Subject:        parts[7],
			Body:           body,
		})
	}
	return commits, nil
}

func (g *Git) CreateTag(ctx context.Context, commit, tag string, opts vcs.CommitOpts) error {
	return nil
}

func (g *Git) DeleteTag(ctx context.Context, commit, tag string) error {
	return nil
}

func (g *Git) ReadTags(ctx context.Context, query string) ([]string, error) {
	args := []string{
		"tag",
	}
	if query != "" {
		args = append(args, "-l", query)
	}
	b, err := g.call(ctx, args)
	if err != nil {
		return nil, err
	}
	var tags []string
	scanner := bufio.NewScanner(bytes.NewBuffer(b))
	for scanner.Scan() {
		s := scanner.Text()
		tags = append(tags, s)
	}
	return tags, nil
}
