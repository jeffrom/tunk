# tunk

![tunk logo](tunk.png)

tunk is an automation tool for tagging releases using a trunk-based development workflow. tunk is unlike some other [Semantic Versioning](https://semver.org/) release tools in that it is intended for trunk-based application release, as opposed to branch-based library releases.

There are several similar (and great) tools that serve a similar purpose, such as [semantic-release](https://github.com/semantic-release/semantic-release). semantic-release is intended for branch-based development, and has different release policies than tunk. With semantic-release, in order to publish a release candidate, you typically need an additional branch. tunk creates another tag on the main branch.

## install

```bash
$ go install github.com/jeffrom/tunk/cmd/tunk
```

## usage

```bash
tunk [rc]

A utility for creating Semantic Version-compliant tags.

FLAGS
      --all                         operate on all scopes
  -b, --branch stringArray          set release branches (default [main,master])
  -c, --config string               specify config file
  -n, --dry-run                     Don't do destructive operations
  -h, --help                        show help
      --major                       bump major version
      --minor                       bump minor version
      --patch                       bump patch version
      --policy stringArray          declare commit policies (default [conventional-lax,lax])
  -q, --quiet                       print as little as necessary
      --release-scope stringArray   declare release scopes
  -s, --scope string                Operate on a scope
      --template string             go text/template for tag
  -v, --verbose                     print additional debugging info
  -V, --version                     print version and exit

EXAMPLES

# bump the version, if there are any new commits
$ tunk

# bump the minor version regardless of the state of the branch.
$ tunk --minor

# bump the version for scope "myscope" only
$ tunk -s myscope

# bump the version for all release scopes (can be defined in tunk.yaml)
$ tunk --all --release-scope myscope --release-scope another-scope
```
