package runner

import (
	"context"
	"io"
	"text/template"

	"github.com/jeffrom/tunk/commit"
)

const defaultShortlogTemplate = `{{ or .Version.Scope .Name "release" }}: v{{ .Version.Version }}

This release contains the following commits:
{{ range $commit := .Version.ScopedCommits }}
* {{ $commit.Subject }} ({{ $commit.ShortID }})
{{ end }}
{{ messageInfo }}
`

type shortlogData struct {
	Version *commit.Version
	Name    string
}

var funcMap = template.FuncMap{
	"messageInfo": func() string {
		return `# Please enter the message for your changes. Lines starting with
# '#' will be ignored.
#
# An empty message does NOT abort the commit.
# ------------------------ >8 ------------------------
# Do not modify or remove the line above.
# Everything below it will be ignored.
`
	},
}

func (r *Runner) shortlog(ctx context.Context, w io.Writer, ver *commit.Version, name string) error {
	if ver == nil {
		return nil
	}
	tmpl := defaultShortlogTemplate
	if r.cfg.LogTemplate != "" {
		tmpl = r.cfg.LogTemplate
	}
	t, err := template.New("shortlog").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(w, shortlogData{Version: ver, Name: name})
}
