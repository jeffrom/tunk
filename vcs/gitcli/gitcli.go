// Package gitcli implements vcs.Interface using the git commandline tool.
package gitcli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jeffrom/tunk/config"
	"github.com/jeffrom/tunk/model"
	"github.com/jeffrom/tunk/vcs"
	"github.com/mattn/go-isatty"
)

// Git implements vcs.Interface using the git commandline tool.
type Git struct {
	cfg            config.Config
	wd             string
	askpass        string
	askpassCleanup func()
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

func (g *Git) GetMainBranch(ctx context.Context, candidates []string) (string, error) {
	if g.cfg.InCI {
		if err := g.setupAskpass(); err != nil {
			return "", err
		}
	}
	if len(candidates) == 0 {
		remoteInfo, err := g.call(ctx, []string{"remote", "show", "origin"})
		if err != nil {
			return "", err
		}
		return getRemoteShowHeadBranch(remoteInfo)
	}
	args := []string{"branch", "--list"}
	for _, cand := range candidates {
		b, err := g.call(ctx, append(args, cand))
		if err != nil {
			return "", err
		}
		match, err := checkListBranchOutput(b, cand)
		if err != nil {
			return "", err
		}
		if match {
			return cand, nil
		}
	}
	return "", fmt.Errorf("no matching release branch of candidates: %q", candidates)
}

func (g *Git) CurrentCommit(ctx context.Context) (string, error) {
	args := []string{"rev-parse", "HEAD"}
	b, err := g.call(ctx, args)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(b)), nil
}

func (g *Git) BranchContains(ctx context.Context, commit, branch string) (bool, error) {
	args := []string{"branch", "--contains", commit, "--list", branch}
	b, err := g.call(ctx, args)
	if err != nil {
		return false, err
	}
	return checkListBranchOutput(b, branch)
}

func (g *Git) CurrentBranch(ctx context.Context) (string, error) {
	args := []string{"branch", "--show-current"}
	b, err := g.call(ctx, args)
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(b)), nil
}

func (g *Git) Push(ctx context.Context, upstream, ref string, opts vcs.PushOpts) error {
	// if g.cfg.InCI {
	// 	// check token, creds, setup author etc
	// }

	args := []string{"push"}
	if opts.Tags {
		args = append(args, "--tags")
	}
	if opts.FollowTags {
		args = append(args, "--follow-tags")
	}
	if g.cfg.InCI {
		args = append(args, "--atomic")
	}
	if upstream == "" {
		upstream = "origin"
	}
	args = append(args, upstream, ref)

	argsStr := ArgsString(args)
	if g.cfg.Dryrun {
		g.cfg.Printf("+ git %s (dryrun)", argsStr)
		return nil
	}
	g.cfg.Printf("+ git %s", argsStr)
	_, err := g.call(ctx, args)
	return err
}

const expectedLogParts = 10

func (g *Git) ReadCommits(ctx context.Context, query string) ([]*model.Commit, error) {
	// TODO chunk the read. use --max-count and the commit id as a cursor
	args := []string{
		"log", "--pretty=tformat:_START_%H_SEP_%aN_SEP_%ae_SEP_%ai_SEP_%cN_SEP_%ce_SEP_%ci_SEP_%s_SEP_%s_SEP_%b_END_", query,
	}
	b, err := g.call(ctx, args)
	if err != nil {
		return nil, err
	}

	var commits []*model.Commit
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		s := scanner.Text()
		parts := strings.Split(s, "_SEP_")
		if len(parts) != expectedLogParts {
			return nil, fmt.Errorf("gitcli: expected %d parts from git log, got %d", expectedLogParts, len(parts))
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
			Ref:            parts[8],
			Body:           body,
		})
	}
	return commits, nil
}

func (g *Git) CreateTag(ctx context.Context, commit, tag string, opts vcs.TagOpts) error {
	if opts.Message == "" {
		opts.Message = tag
	}
	if g.cfg.InCI && (opts.Author == "" || opts.AuthorEmail == "") {
		g.cfg.Printf("CI: setting author, author email")
		// TODO these should be configurable via flags, env vars, etc
		opts.Author = "tunk"
		opts.AuthorEmail = "cool+release@example.com"
	}
	if g.cfg.InCI {
		if opts.Author != "" || opts.AuthorEmail != "" {
			if err := g.setAuthor(ctx, opts.Author, opts.AuthorEmail); err != nil {
				return err
			}
		}
		if err := g.setupAskpass(); err != nil {
			return err
		}
	}

	tmpfile, err := ioutil.TempFile("", "tunk-shortlog")
	if err != nil {
		return err
	}
	defer tmpfile.Close()
	defer os.RemoveAll(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(opts.Message)); err != nil {
		return err
	}

	args := []string{
		"tag", "-a", tag,
	}
	if commit != "" {
		args = append(args, commit)
	}
	stdoutfd := os.Stdout.Fd()
	istty := isatty.IsTerminal(stdoutfd)
	if !g.cfg.InCI && !g.cfg.NoEdit && istty {
		args = append(args, "-e")
	}
	args = append(args, "-F", tmpfile.Name())

	if g.cfg.Dryrun {
		g.cfg.Printf("+ git %s (dryrun)", ArgsString(args))
		return nil
	}
	cmd := CommandContext(ctx, "git", args...)
	cmd.Dir = g.wd
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return err
}

func (g *Git) DeleteTag(ctx context.Context, commit, tag string) error {
	return errors.New("not implemented")
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
	userArgs := []string{"config", "--local", "user.name", author}
	emailArgs := []string{"config", "--local", "user.email", email}
	if g.cfg.Dryrun {
		g.cfg.Printf("+ git %s (dryrun)", ArgsString(userArgs))
		g.cfg.Printf("+ git %s (dryrun)", ArgsString(emailArgs))
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

func (g *Git) setupAskpass() error {
	if g.askpass != "" || !g.cfg.InCI {
		return nil
	}
	token := getenv("GIT_TOKEN", "GITHUB_TOKEN", "GH_TOKEN")
	if token == "" {
		return errors.New("gitcli tag: GIT_TOKEN, GITHUB_TOKEN, or GH_TOKEN is required for CI mode")
	}

	// TODO cleanup
	askpass, cleanup, err := SetupCreds()
	if err != nil {
		return err
	}

	g.askpass = askpass
	g.askpassCleanup = cleanup
	return nil
}

func (g *Git) Cleanup() error {
	if g.askpassCleanup != nil {
		g.askpassCleanup()
	}
	return nil
}

var (
	// git@github.com:jeffrom/tunk.git
	gitRemoteSSHURLRE = regexp.MustCompile(`^(?P<prefix>[^@]+[^:]+:)?(?P<path>.+)$`)

	// https://cool:whatever@github.com/jeffrom/tunk.git
	gitRemoteHTTPURLRE = regexp.MustCompile(`^(?P<prefix>https?://[^/]+/)(?P<path>.+)$`)
)

func (g *Git) ReadNameFromRemoteURL(ctx context.Context, upstream string) (string, error) {
	if upstream == "" {
		upstream = "origin"
	}
	args := []string{"config", "--get", fmt.Sprintf("remote.%s.url", upstream)}
	b, err := g.call(ctx, args)
	if err != nil {
		return "", vcs.ErrRemoteUnavailable
	}
	rawURI := strings.TrimSpace(string(b))

	m := gitRemoteSSHURLRE.FindStringSubmatch(rawURI)
	if m == nil {
		m = gitRemoteHTTPURLRE.FindStringSubmatch(rawURI)
	}

	if m == nil {
		return "", fmt.Errorf("failed to parse upstream url %q", rawURI)
	}

	uri, err := url.Parse(m[2])
	if err != nil {
		return "", err
	}

	_, suff := filepath.Split(filepath.Clean(uri.Path))
	parts := strings.SplitN(suff, ".", 2)

	return parts[0], nil
}

func noop() {}

func SetupCreds() (string, func(), error) {
	token := getenv("GIT_TOKEN", "GITHUB_TOKEN", "GH_TOKEN")
	f, err := ioutil.TempFile("", "tunk")
	if err != nil {
		return "", noop, err
	}
	defer f.Close()

	var storeBuilder strings.Builder
	storeBuilder.WriteString(`echo "`)
	if token != "" {
		storeBuilder.WriteString(token)
	}
	storeBuilder.WriteString(`"`)

	b := []byte(storeBuilder.String())
	if _, err := f.Write(b); err != nil {
		return "", noop, err
	}

	if err := os.Chmod(f.Name(), 0700); err != nil {
		return "", noop, err
	}

	return f.Name(), func() {
		os.Remove(f.Name())
	}, nil
}

// func (g *Git) setUpstream(ctx context.Context, upstream, repoName, remoteName, user, token string) error {
// 	printSuffix := ""
// 	if g.cfg.Dryrun {
// 		printSuffix = " (dryrun)"
// 	}
// 	scrubbedURL := fmt.Sprintf("https://%s:xxxxxx@github.com/%s/%s.git", user, repoName, remoteName)
// 	url := fmt.Sprintf("https://%s:%s@github.com/%s/%s.git", user, token, repoName, remoteName)
// 	b, err := g.call(ctx, []string{"remote", "get-url", upstream})
// 	currURL := strings.TrimSuffix(string(b), "\n")
// 	if err != nil {
// 		args := []string{"remote", "add", upstream}
// 		g.cfg.Printf("+ git %s%s", ArgsString(append(args, scrubbedURL)), printSuffix)
// 		if g.cfg.Dryrun {
// 			return nil
// 		}
// 		_, aerr := g.call(ctx, append(args, url))
// 		return aerr
// 	} else if currURL != url {
// 		args := []string{"remote", "set-url", upstream}
// 		g.cfg.Printf("+ git %s%s", ArgsString(append(args, scrubbedURL)), printSuffix)
// 		if g.cfg.Dryrun {
// 			return nil
// 		}
// 		_, serr := g.call(ctx, append(args, url))
// 		return serr
// 	}
// 	return nil
// }

func checkListBranchOutput(out []byte, candidate string) (bool, error) {
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		line := strings.Trim(s.Text(), " \t*")
		if line == candidate {
			return true, nil
		}
	}
	if err := s.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func getenv(names ...string) string {
	for _, name := range names {
		if env := os.Getenv(name); env != "" {
			return env
		}
	}
	return ""
}

// HEAD branch: master
var remoteShowHeadBranchRE = regexp.MustCompile(`^\s*HEAD branch: (?P<branch>\S+)$`)

func getRemoteShowHeadBranch(b []byte) (string, error) {
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		m := remoteShowHeadBranchRE.FindSubmatch(s.Bytes())
		if len(m) == 2 {
			return string(m[1]), nil
		}
	}
	if err := s.Err(); err != nil {
		return "", err
	}
	return "", nil
}
