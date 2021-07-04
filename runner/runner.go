// Package runner manages command-line execution
package runner

import (
	"bytes"
	"context"
	"fmt"

	"github.com/jeffrom/tunk/commit"
	"github.com/jeffrom/tunk/config"
	"github.com/jeffrom/tunk/vcs"
)

type Runner struct {
	cfg        config.Config
	vcs        vcs.Interface
	analyzer   *commit.Analyzer
	tag        *commit.Tag
	mainBranch string
}

func New(cfg config.Config, vcs vcs.Interface) (*Runner, error) {
	tag, err := commit.NewTag(cfg.TagTemplate)
	if err != nil {
		return nil, err
	}
	return &Runner{
		cfg:      cfg,
		vcs:      vcs,
		tag:      tag,
		analyzer: commit.NewAnalyzer(cfg, vcs, tag),
	}, nil
}

// Check checks initial requirements for release, such as being on the right branch.
func (r *Runner) Check(ctx context.Context, rc string) error {
	// TODO detect scopes from tags (check first release_scopes /
	// allowed_scopes, and fall back to detection) and check them all
	if r.mainBranch == "" {
		branches := r.cfg.Branches
		if r.cfg.InCI && !r.cfg.BranchesSet {
			branches = nil
		}
		var mainBranch string
		var err error
		mainBranch, err = r.vcs.GetMainBranch(ctx, branches)
		if err != nil {
			r.cfg.Printf("Get remote failed, falling back to defaults: %v", r.cfg.Branches)
			mainBranch, err = r.vcs.GetMainBranch(ctx, r.cfg.Branches)
			if err != nil {
				return err
			}
		}
		r.mainBranch = mainBranch
		r.cfg.Printf("Main branch is %q", mainBranch)
	}

	currBranch, err := r.vcs.CurrentBranch(ctx)
	if err != nil {
		return err
	}
	if currBranch != r.mainBranch && !r.cfg.Dryrun {
		return wrongBranchError{mainBranch: r.mainBranch, branch: currBranch}
	}
	return nil
}

func (r *Runner) Analyze(ctx context.Context, rc string) ([]*commit.Version, error) {
	return r.analyzer.Analyze(ctx, rc)
}

func (r *Runner) CreateTags(ctx context.Context, versions []*commit.Version) error {
	name := r.cfg.Name
	if name == "" {
		var err error
		name, err = r.vcs.ReadNameFromRemoteURL(ctx, "")
		if err != nil {
			return err
		}
	}

	for _, ver := range versions {
		opts := vcs.TagOpts{}
		tag, err := RenderTag(r.cfg, r.tag, ver)
		if err != nil {
			return err
		}
		r.cfg.Printf("creating tag %q for commit %s...", tag, ver.ShortCommit())

		b := &bytes.Buffer{}
		if err := r.shortlog(ctx, b, ver, name); err != nil {
			return err
		}
		shortlog := b.String()
		opts.Message = shortlog
		if r.cfg.Dryrun {
			r.cfg.Printf("shortlog:\n\n---\n%s", shortlog)
		} else {
			r.cfg.Debugf("shortlog:\n\n---\n%s", shortlog)
		}

		if err := r.vcs.CreateTag(ctx, ver.Commit, tag, opts); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) PushTags(ctx context.Context) error {
	if err := r.vcs.Push(ctx, "origin", r.mainBranch, vcs.PushOpts{FollowTags: true}); err != nil {
		return err
	}
	return nil
}

func RenderTag(cfg config.Config, t *commit.Tag, ver *commit.Version) (string, error) {
	return t.ExecuteString(commit.TagData{Version: ver})
}

type wrongBranchError struct {
	mainBranch string
	branch     string
}

func (e wrongBranchError) Error() string {
	return fmt.Sprintf("commit must be on branch %s, not %s", e.mainBranch, e.branch)
}

func isWrongBranchError(err error) bool {
	_, ok := err.(wrongBranchError)
	return ok
}
