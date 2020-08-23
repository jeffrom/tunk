// Package gitcli implements vcs.Interface using the git commandline tool.
package gitcli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jeffrom/trunk-release/config"
	"github.com/jeffrom/trunk-release/model"
	"github.com/jeffrom/trunk-release/vcs"
)

// Git implements vcs.Interface using the git commandline tool.
type Git struct {
	cfg config.Config
	wd  string
}

func New(cfg config.Config, wd string) *Git {
	return &Git{
		cfg: cfg,
		wd:  wd,
	}
}

func (g *Git) Fetch(ctx context.Context, upstream, ref string) error {
	return nil
}

func (g *Git) Push(ctx context.Context, upstream, ref string, opts vcs.PushOpts) error {
	if g.cfg.InCI {
		// check token, creds, setup author etc
	}

	args := []string{"push"}
	if opts.FollowTags {
		args = append(args, "--follow-tags")
	}
	if upstream == "" {
		upstream = "origin"
	}
	args = append(args, upstream, ref)

	if g.cfg.Dryrun {
		g.cfg.Printf("+ git %s (dryrun)", argsString(args))
		return nil
	}
	_, err := g.call(ctx, args)
	return err
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

func (g *Git) CreateTag(ctx context.Context, commit, tag string, opts vcs.TagOpts) error {
	if opts.Message == "" {
		return errors.New("gitcli: message is required")
	}
	if g.cfg.InCI && (opts.Author == "" || opts.AuthorEmail == "") {
		g.cfg.Printf("CI: setting author, author email")
		opts.Author = "trunk-release"
		opts.AuthorEmail = "cool+release@example.com"
	}
	if g.cfg.InCI {
		if opts.Author != "" || opts.AuthorEmail != "" {
			if err := g.setAuthor(ctx, opts.Author, opts.AuthorEmail); err != nil {
				return err
			}
		}
		ghToken := os.Getenv("GITHUB_TOKEN")
		if ghToken == "" {
			return errors.New("gitcli tag: GITHUB_TOKEN is required in CI")
		}
	}

	args := []string{
		"tag", "-a", tag,
	}
	if opts.Commit != "" {
		args = append(args, opts.Commit)
	}
	args = append(args, "-m", opts.Message)

	if g.cfg.Dryrun {
		g.cfg.Printf("+ git %s (dryrun)", argsString(args))
		return nil
	}
	_, err := g.call(ctx, args)
	return err
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

func (g *Git) setAuthor(ctx context.Context, author, email string) error {
	userArgs := []string{"config", "user.name", author}
	emailArgs := []string{"config", "user.email", email}
	if g.cfg.Dryrun {
		g.cfg.Printf("+ git %s (dryrun)", argsString(userArgs))
		g.cfg.Printf("+ git %s (dryrun)", argsString(emailArgs))
		return nil
	}
	if _, err := g.call(ctx, userArgs); err != nil {
		return err
	}
	if _, err := g.call(ctx, emailArgs); err != nil {
		return err
	}
	return nil
}

func (g *Git) setUpstream(ctx context.Context, upstream, repoName, remoteName, user, token string) error {
	printSuffix := ""
	if g.cfg.Dryrun {
		printSuffix = " (dryrun)"
	}
	scrubbedURL := fmt.Sprintf("https://%s:xxxxxx@github.com/%s/%s.git", user, repoName, remoteName)
	url := fmt.Sprintf("https://%s:%s@github.com/%s/%s.git", user, token, repoName, remoteName)
	b, err := g.call(ctx, []string{"remote", "get-url", upstream})
	currURL := strings.TrimSuffix(string(b), "\n")
	if err != nil {
		args := []string{"remote", "add", upstream}
		g.cfg.Printf("+ git %s%s", argsString(append(args, scrubbedURL)), printSuffix)
		if g.cfg.Dryrun {
			return nil
		}
		_, aerr := g.call(ctx, append(args, url))
		return aerr
	} else if currURL != url {
		args := []string{"remote", "set-url", upstream}
		g.cfg.Printf("+ git %s%s", argsString(append(args, scrubbedURL)), printSuffix)
		if g.cfg.Dryrun {
			return nil
		}
		_, serr := g.call(ctx, append(args, url))
		return serr
	}
	return nil
}
