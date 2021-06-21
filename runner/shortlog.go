package runner

import (
	"context"
	"io"
	"text/template"

	"github.com/jeffrom/tunk/commit"
)

const defaultShortlogTemplate = `release: v{{ .Version.Version }}

This release contains the following commits:
{{ range $commit := .Version.AllCommits }}
* {{ $commit.Subject }} ({{ $commit.ShortID }})
{{ end }}`

type shortlogData struct {
	Version *commit.Version
}

func (r *Runner) shortlog(ctx context.Context, w io.Writer, ver *commit.Version) error {
	if ver == nil {
		return nil
	}
	tmpl := defaultShortlogTemplate
	if r.cfg.LogTemplatePath != "" {
		tmpl = r.cfg.LogTemplatePath
	}
	t, err := template.New("shortlog").Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(w, shortlogData{Version: ver})
}
