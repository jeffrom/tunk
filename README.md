# tunk

tunk is an automation tool for tagging releases using a trunk-based development workflow. tunk is unlike some other [Semantic Versioning](https://semver.org/) release tools in that it is intended for trunk-based application release strategies where there is one main trunk branch, and all other branches tend to be short-lived.

There are several similar (and great) tools that serve a similar purpose, such as [semantic-release](https://github.com/semantic-release/semantic-release). semantic-release is intended for branch-based development, and has different release policies than tunk. With semantic-release, in order to publish a release candidate, you typically need an additional branch. tunk just creates another tag on the main branch.

## status

😬 *not super stable* 😬

tunk is pretty stable, but also has bugs, and I may introduce some breaking
changes until tunk reaches `v1.0.0`. After that, breaking changes are very
unlikely.

## install

tunk depends on git being available in the system $PATH.

From source:

```bash
$ git clone jeffrom/tunk
$ make
$ sudo make install  # uninstall with sudo make uninstall
```

Install using go toolchain:

```bash
$ go install github.com/jeffrom/tunk/cmd/tunk
```

Install the latest version directly from github using curl:

```bash
$ export version=$(curl --silent "https://api.github.com/repos/jeffrom/tunk/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
$ curl -L -o tunk.tar.gz https://github.com/jeffrom/tunk/releases/download/$version/tunk_$(echo -n $version | sed -e 's/^v//')_$(uname -s)_$(uname -p).tar.gz
$ tar zxf tunk.tar.gz
$ mv tunk /usr/local/bin/
```

## usage

On a git repository with no tags, running `tunk` will print a message that no release tags could be found, and steps for creating one. In the default configuration, the initial tag will look like this: `v0.1.0`. Once a tag has been created, subsequent commits can be automatically tagged.

Running `tunk` on a repository with matching tags will open `$EDITOR` (or `$GIT_EDITOR`) with a summary of the commits that comprise the pending release for final edits. Once saved, a tag will be created for the current commit.

Some example usages:

Create a new release tag:

```bash
$ tunk
```

See what the next tag will be:

```bash
$ tunk -n
```

Only print the next version (useful for scripting):

```bash
$ tunk -nq
```

Bump the minor version:

```bash
$ tunk --minor
```

Tag a release candidate named "myrc":

```bash
$ tunk myrc
```

### scopes

Code projects often have multiple release artifacts, and it can be useful to have separate release channels. Scopes provide this by reading it from the commit message. For example, if we had a go project with a main module defined at the git repository root, and another nested somewhere in the directory tree, a release of the sub-module could be executed by running `tunk -s mymodule`. In the default configuration, this would create a tag like `mymodule/v1.2.3`, which is compatible with go mod.

### policies

Tags versions are decided using a set of "policies." The default policies are:

1. A conventional commit policy with the following type mappings:

```
feat:        MINOR
fix:         PATCH
revert:      PATCH
cont:        PATCH
perf:        PATCH
improvement: PATCH
refactor:    PATCH
style:       PATCH
test:        SKIP
chore:       SKIP
docs:        SKIP
```

2. fallback to a lax policy where any commit triggers a patch bump

Custom policies can be defined in a projects directory root, or in any parent directory, in a file called `tunk.yaml`.

The default policy configuration triggers a release for all commits except `test`, `chore`, and `docs`, which is a reasonably low-friction way to release. Custom policies can also be defined in `tunk.yaml`. It's also possible to disable one or both of the default policies. If no policies match any commits, or no policies are set, tunk will fail (unless an override flag, such as `--minor`, is provided). An easy way to require a manual override is to run `tunk --no-policy` (or set `policies: []` in tunk.yaml).

### templates

tunk can read and write tags according to a template. See `tunk --help` for more information.

### release candidates

Prerelease versions can be released on the main branch in additional to regular releases. For example, running `tunk rc` will create tag `v1.2.3-rc.0`. If tunk is called again with the same arguments on a later commit (that results in the same version `v1.2.3`), it will be tagged `v1.2.3-rc.1`, and so on. tunk ignores the build metadata portion of semver strings.

### validation mode

tunk can be run in validation mode, which will print any invalid commits, taking into account allowed commit types and scopes, as well as configured policies. To run it against all commits since the last release, use: `tunk --check`. To check subjects only, use `tunk --check-commit "my commit subject"`, or `echo "my commit subject" | tunk --check-commit -`.

### continuous integration

tunk runs easily in continuous integration systems, either by running `tunk --ci`, or if the `$CI` environment variable is set to "true", "1", or "yes". In CI mode, tunk will push the tags it creates. tunk should work with any git server that supports password authentication. SSH should work as well, as long as its configured correctly (ie the ssh key is passwordless, and the host is authorized).

## configuration

tunk can be configured via command-line flags, a YAML configuration file, and, in some cases, environment variables. An example configuration file, which contains the default configuration, is located at [testdata/tunk.example.yaml](testdata/tunk.example.yaml).

### environment variables

The following environment variables configure tunk:

- `CI` - puts tunk in CI mode
- `GIT_TOKEN`, `GITHUB_TOKEN`, `GH_TOKEN` - sets the git password in CI mode

## documentation

tunk has man pages. To view them:

```bash
$ man tunk
$ man tunk-ci
$ man 5 tunk-config
```

## inspirations

* [semantic-release](https://github.com/semantic-release/semantic-release)
* [semver](https://git.sr.ht/~sircmpwn/dotfiles/tree/master/bin/semver) by Drew DeVault
