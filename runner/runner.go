// Package runner manages command-line execution
package runner

import (
	"context"

	"github.com/jeffrom/tunk/commit"
	"github.com/jeffrom/tunk/config"
	"github.com/jeffrom/tunk/vcs"
)

type Runner struct {
	cfg      config.Config
	vcs      vcs.Interface
	analyzer *commit.Analyzer
}

func New(cfg config.Config, vcs vcs.Interface) *Runner {
	return &Runner{
		cfg:      cfg,
		vcs:      vcs,
		analyzer: commit.NewAnalyzer(cfg, vcs),
	}
}

func (r *Runner) Analyze(ctx context.Context, rc string) ([]*commit.Version, error) {
	return r.analyzer.Analyze(ctx, rc)
}

func (r *Runner) CreateTags(ctx context.Context, versions []*commit.Version) error {
	for _, ver := range versions {
		opts := vcs.TagOpts{}
		tag := ver.GitTag()
		r.cfg.Printf("creating tag %q for commit %s", tag, ver.ShortCommit())
		if err := r.vcs.CreateTag(ctx, ver.Commit, tag, opts); err != nil {
			return err
		}
	}
	return nil
}
