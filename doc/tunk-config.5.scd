tunk-config(5)

# NAME

tunk-config - configuration options for *tunk*(1)

# CONFIGURATION

TODO explain the configuration file behavior of tunk

TODO explain how to see example tunk.yamls

TODO explain the format: yaml (the json equivalent part) or json

# OPTIONS

TODO list all the available options

# POLICIES

TODO explain how policies work, how they can be defined and stacked, fallbacks,
etc

# TEMPLATING

Tags can be created and read according to a template. This is provided by the Go
text/template package. See "go doc text/template" for more information.

The following template data is available for tag templates:

XXX

The following functions are available in templates:

[[ *Name*
:- *Description*
|  join
:- strings.Join

++
See "go doc github.com/jeffrom/tunk/commit TagData" for more information.

In order for templates to be able to correctly read and create tags, the only
requirement is that a SemVer-compliant version is included as one continuous
string, including the prerelease portion.

## EXAMPLES

The default tag template is roughly:

```
{{- with $scope := .Version.Scope -}}{{- $scope -}}/{{- end -}}
v{{- semver .Version -}}
```

As an example of a custom template, here's one that doesn't include the "v"
prefix, and changes the scope delineator from "/" to "#":

```
{{- with $scope := .Version.Scope -}}{{- $scope -}}#{{- end -}}
{{- semver .Version -}}
```

## SHORTLOG

The tag message is also rendered from a template which can be overridden. The
default template is roughly:

```
{{ or .Version.Scope .Name "release" }}: v{{ .Version.Version }}

This release contains the following commits:
{{ range $commit := .Version.AllCommits }}
* {{ $commit.Subject }} ({{ $commit.ShortID }})
{{ end }}
```

# SEE ALSO

# AUTHORS