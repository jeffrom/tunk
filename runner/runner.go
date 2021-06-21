// Package runner manages command-line execution
package runner

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/jeffrom/tunk/commit"
	"github.com/jeffrom/tunk/config"
	"github.com/jeffrom/tunk/vcs"
)

type Runner struct {
	cfg        config.Config
	vcs        vcs.Interface
	analyzer   *commit.Analyzer
	mainBranch string
}

func New(cfg config.Config, vcs vcs.Interface) *Runner {
	return &Runner{
		cfg:      cfg,
		vcs:      vcs,
		analyzer: commit.NewAnalyzer(cfg, vcs),
	}
}

func (r *Runner) Check(ctx context.Context, rc string) error {
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
		return fmt.Errorf("commit must be on branch %s, not %s", r.mainBranch, currBranch)
	}
	return nil
}

func (r *Runner) Analyze(ctx context.Context, rc string) ([]*commit.Version, error) {
	return r.analyzer.Analyze(ctx, rc)
}

func (r *Runner) CreateTags(ctx context.Context, versions []*commit.Version) error {
	for _, ver := range versions {
		opts := vcs.TagOpts{}
		tag, err := RenderTag(r.cfg, ver)
		if err != nil {
			return err
		}
		r.cfg.Printf("creating tag %q for commit %s", tag, ver.ShortCommit())
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

func RenderTag(cfg config.Config, ver *commit.Version) (string, error) {
	var tag string
	var tmpl *template.Template
	if cfg.TagTemplate != "" {
		cfg.Debugf("custom tag template: %q", cfg.TagTemplate)
		// TODO cache this
		var err error
		tmpl, err = template.New("custom_tag").Parse(cfg.TagTemplate)
		if err != nil {
			return "", err
		}
		b := &bytes.Buffer{}
		if err := tmpl.Execute(b, ver); err != nil {
			return "", err
		}
		tag = b.String()
	} else {
		tag = ver.GitTag()
	}

	return tag, nil
}
