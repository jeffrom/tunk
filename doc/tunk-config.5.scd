tunk-config(5)

# NAME

tunk-config - configuration options for *tunk*(1)

# CONFIGURATION

By default, a configuration file, *tunk.yaml* will be searched for in the
current or any parent directory. The file can also be specified using the _-c_
flag. For each configuration variable, the configuration file will be evaluated
first, then environment variables, then flags.

To view the default configuration, run: *tunk --print-default-config*.

The format for configuration files is YAML. JSON is also supported.

# OPTIONS

*ci*
	Always operate in CI mode.

	Default: false

*quiet*
	Print only errors, or if in --dry-run mode, print the next tag only.

	Default: false

*branches*
	Allowed release branches.

	Default: Detected using repository metadata

*release_scopes*
	Create tags for the specified scopes

	Default: []

*policies*
	Declare policy or policies by name. To require manual version bumping, set
	_policies: []_.

	See POLICIES section for more information.

	Default: [conventional-lax, lax]

*custom_policies*
	Define custom policies. See POLICIES section for more information.

*tag_template*
	Define custom tag template. See TEMPLATING section for more information.

*log_template*
	Define custom shortlog template. See TEMPLATING section for more information.

*no_edit*
	Skip final message edits before creating the tag.

	Default: false

*allowed_scopes*
	If defined, causes *tunk --check* to fail if a scope a policy parses a scope
	not in this list. To allow the main scope, use an empty string.

	Default: []

*allowed_types*
	If defined, causes *tunk --check* to fail if a scope a policy parses a
	commit type not in this list.

	Default: []

# POLICIES

Policies can be used to customize parsing and validation of commit messages.
They also allow some customization of decision behavior of what release type to
attach to commits.

Multiple policies can be declared, in which case tunk will use the first
matching policy to decide the release type for a given commit.

The flavor of regular expressions in *tunk* is go regular expressions. See *go
doc regexp/syntax* for more information.

The policy engine is built to read commits according to the Convention Commits
v1.0.0 spec, but other formats can also be used.

See _https://www.conventionalcommits.org/en/v1.0.0/_.

The following configuration variables are available for custom policies:

*name*
	The name of the policy.

*subject_regex*
	A regular expression to match commit subjects. The following named capture
	groups, if matched, will be used by the policy engine:

	- type: Conventional Commits _Type_.
	- scope: Conventional Commits _Scope_.

*body_annotation_start_regex*
	A regular expression to read body annotations, such as _BREAKING CHANGE_.
	The following named capture groups, if matched, will be used by the policy
	engine:

	- *name*: The name of the body annotation.

*breaking_change_annotations*
	A list of strings that, if matched by the body annotation *name*, cause a
	commit to be marked as a breaking (major version) change.

*commit_types*
	A map of commit subject _types_ to the release type to tag the commit with.
	The following release types are supported:

	- PATCH
	- MINOR
	- MAJOR
	- SKIP

*fallback_type*
	A release type that will be the final type attached to a commit, if no other
	policies match. A release type of _SKIP_ is functionally a noop.

See *tunk(1)* for available built-in policies.

# TEMPLATING

Tags can be created and read according to a template. This is provided by the Go
text/template package. Templates are used to generate the git glob query, as
well as the tag itself. This is accomplished using a naive solution of replacing
the template version data with an asterisk.

See *go doc text/template* (or _https://golang.org/pkg/text/template/_) for more
information about the template format.

The following template data is available for tag templates:

[[ *Name*
:- *Type*
:- *Description*
|  Version
:-  struct
:- The analyzed version, including the release scope
|  Version.Scope
:-  string
:- The release scope
|  Version.Major
:-  uint64
:- The release's major version
|  Version.Minor
:-  uint64
:- The release's minor version
|  Version.Patch
:-  uint64
:- The release's patch version
|  Version.Pre
:-  []string
:- The prerelease section
|  Version.ShortCommit
:-  string
:- The release commit ID, truncated to 8 bytes

The following functions are available in tag templates:

[[ *Name*
:- *Description*
|  join
:- direct mapping to go stdlib's strings.Join
|  semver
:- renders a SemVer-compliant version string, including prerelease information

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

*tunk*(1), *tunk-ci*(1)

# AUTHORS

Maintained by Jeff Martin <jeffmartin@gmail.com>.
