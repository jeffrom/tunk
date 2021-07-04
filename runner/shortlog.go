package runner

import (
	"context"
	"io"
	"text/template"

	"github.com/jeffrom/tunk/commit"
)

const defaultShortlogTemplate = `{{ or .Version.Scope .Name "release" }}: v{{ .Version.Version }}

This release contains the following commits:
{{ range $commit := .Version.AllCommits }}
* {{ $commit.Subject }} ({{ $commit.ShortID }})
{{ end }}

# Please enter the message for your changes. Lines starting with
# '#' will be ignored.
#
# An empty message does NOT abort the commit.
# ------------------------ >8 ------------------------
# Do not modify or remove the line above.
# Everything below it will be ignored.
`

type shortlogData struct {
	Version *commit.Version
	Name    string
}

func (r *Runner) shortlog(ctx context.Context, w io.Writer, ver *commit.Version, name string) error {
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
	return t.Execute(w, shortlogData{Version: ver, Name: name})
}
