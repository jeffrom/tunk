tunk(1)

# NAME

tunk - a trunk-based git release tool for folks who can't spell

# SYNOPSIS

_tunk_ [-Vhnq]
	\ \[-c _file_]
	\ \[--major|--minor|--patch]
	\ \[--check|--check-commit _subject_]
	\ \[--stats|--stats-all]
	\ \[--policy|--no-policy] [--policy-view]
	\ \[--allowed-scope] [--allowed-type]
	\ \[--release-scope]
	\ \[--scope|--all]
	\ \[--no-edit] [--name]
	\ \[--template] [--shortlog-template]
	\ \[<prerelease>]

# DESCRIPTION

*tunk* is a utility that creates Semantic-Version compliant git tags. It can be
configured to support a number of release policies using a config file, which by
default is read from tunk.yaml in the current directory, or the first parent
directory where it can be found.

It parses commit messages according to policies, which are used to automatically
bump versions and validate commit messages. It can also act as a manual version
bumping utility. When run in a terminal, it will open *$EDITOR* for final
tag message changes before creating the tag.

# OPTIONS

*-V, --version*
	Prints the installed version of tunk and exits.

*-h, --help*
	Prints help and exits.

*-n, --dry-run*
	Prints next version, but does not actually create it.

*-q, --quiet*
	Print only errors, or if in --dry-run mode, print the next tag only.

*-c, --config* _file_
	Specify a configuration file. See *tunk-config*(5).

*--major, --minor, --patch*
	Bump major, minor, or patch version. Ignores any policies.

*-C, --check*
	Check commits since last release according to configured release policies.

*--check-commit*
	Check commit subjects according to configured policies. Can be specified
	multiple times, unless *"-"* is supplied, in which case newline-terminated
	subjects are read from *stdin*.

*-S, --stats*
	Print basic stats about the repository and exit, including only the top ten
	results for each counter.

*-A, --stats-all*
	Print basic stats about the repository and exit, including all counters.

*-E, --no-edit*
	Skip final message edits before creating the tag.

*--scope* _scope_
	Operate on a single _scope_.

*--all*
	Operate on all scopes. Multiple tags can be created in this mode.

*--policy*
	Declares a commit policy. Can be specified multiple times. Default:
	*[conventional-lax, lax]*

*--no-policy*
	Disables all policies. When this flag is set, one of the *--major*, *--minor*, or
	*--patch* flags is required.

*--policy-view*
	Print some information about the named policies, then exit. Uses whichever
	policies are configured.

*--release-scope* _scope_
	Declares a release scope for the project. If commits that include _scope_
	are read according to a policy, a tag will be created for the latest scoped
	commit. Can be specified multiple times.

*--allowed-scope* _scope_
	Declares an allowed scope for the project. If commits that don't include _scope_
	are read according to a policy, *tunk --check* will fail. Can be specified
	multiple times.

*--allowed-type* _type_
	Declares an allowed commit type for the project. If commits that don't
	include _type_ are read according to a policy, *tunk --check* will fail. Can
	be specified multiple times.

*--template*
	Specify a custom template for rendering tags and tag queries. See
	*tunk-config*(5) for more information on templating.

*--shortlog-template*
	Specify a custom template for rendering the release message. See
	*tunk-config*(5) for more information.

*--name*
	Provide the name of the root project. If this is not provided, tunk falls
	back to reading it from *git config --get remote.origin.url*.

*--latest*
	Prints the latest version in the repository, filtering for prereleases, then
	exits.

*prerelease*
	Create a prerelease tag.

# CONFIGURATION

A configuration file can be specified using the *-c* flag. Otherwise,
configuration is read from tunk.yaml found in the current directory or any
parent.

Configuration file options are overridden by command-line flags.

For more information, see *tunk-config*(5).

# TAG FORMAT

In the default configuration, tags are created with the following format:

```
vM.m.p
```

Tags can also include scopes and prerelease information. For example, a tag
version v.1.2.3, with a scope named "cool", and a prerelease named "burrito",
*tunk burrito -s cool* would create this tag:

```
cool/v.1.2.3-burrito.0
```

If an additional patch commit was then created, and the same command was run
again, the following tag would be created:

```
cool/v.1.2.3-burrito.1
```

and so on, until a new minor or major prerelease version, or a new release
version is created, at which point that counter will restart at zero.

## SCOPES

Releases can be scoped using a name, which in the default configuration will
create a tag with the following format:

```
SCOPE/vM.m.p
```

## PRERELEASES

Semantic-Version prelease tags can be created that have the following structure,
where and *RC* is the prerelease's name, and *N* is a number incremented for
each new prerelease that already matches a release version:

```
vM.m.p-RC.N
```

# POLICIES

Commits can be parsed and validated according to policies, which can be
specified using *--policy*. If no policy matches a commit, it is ignored for the
purposes of calculating the next version. The following policies are available
by default:

[- *Policy*
:- *Format*
:- *Description*
|  conventional-lax
:[ type(scope): body
:[ A Conventional Commits format with a relaxed set of common types
|  lax
:[ scope: body
:[ A fallback policy that always bumps the patch version

++
The default policies are: *conventional-lax, lax*

For information on configuring custom policies, see *tunk-config*(5).

# CONTINUOUS INTEGRATION

*tunk* will run in CI mode if the *--ci* flag is set, or if the environment
variable *$CI* is set to _1_, _true_, or _yes_.

For more information, see *tunk-ci*(1).

# EXAMPLES

To create a new release tag:

```
$ tunk
```

Create a prerelease tag named "rc":

```
$ tunk rc
```

To require a manual version bump:

```
$ tunk -P  # will fail unless --major, --minor, or --patch are provided
```

# SEE ALSO

*tunk-ci*(1), *tunk-config*(5)

# AUTHORS

Maintained by Jeff Martin <jeffmartin@gmail.com>.
