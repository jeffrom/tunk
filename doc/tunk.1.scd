tunk(1)

# NAME

tunk - a trunk-based git release tool for folks who can't spell

# SYNOPSIS

_tunk_ [-Vhnq]
	\ \[-c _file_]
	\ \[--major|--minor|--patch]
	\ \[--check|--check-commit _subject_]
	\ \[--stats|--stats-all]
	\ \[--no-edit]
	\ \[<prerelease>]

# DESCRIPTION

*tunk* is a utility that creates Semantic-Version compliant git tags. It can be
configured to support a number of release policies using a config file, which by
default is read from tunk.yaml in the current directory, or the first parent
directory where it can be found.

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
	Bump major, minor, or patch version, ignoring any policies.

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

## SCOPES

Releases can be scoped using a name, which in the default configuration will
create a tag with the following format:

```
SCOPE/vM.m.p
```

## PRERELEASE VERSIONS

tunk creates Semantic-Version prelease tags can be created that have the
structure, "-NAME.N", where N is a number incremented for each new prerelease
that already matches a release version.

# POLICIES

Commits can be parsed and validated according to policies, which can be
specified using *--policy*. The following policies are available by default:

[[ *Policy*
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

# SEE ALSO

*tunk-ci*(1), *tunk-config*(5)

# AUTHORS

Maintained by Jeff Martin <jeffmartin@gmail.com>.
