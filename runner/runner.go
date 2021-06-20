// Package runner manages command-line execution
package runner

import (
	"bytes"
	"context"
	"text/template"

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
