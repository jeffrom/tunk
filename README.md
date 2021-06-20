# tunk

![tunk logo](tunk.png)

tunk is an automation tool for tagging releases using a trunk-based development workflow. tunk is unlike some other [Semantic Versioning](https://semver.org/) release tools in that it is intended for trunk-based application release, as opposed to branch-based library releases.

There are several similar (and great) tools that serve a similar purpose, such as [semantic-release](https://github.com/semantic-release/semantic-release). semantic-release is intended for branch-based development, and has more opinionated policies about releases than tunk. Unlike tunk, with semantic-release, in order to publish a release candidate, you typically need to manage a separate branch.

Because tunk has a small scope, it's easy to combine with other release tools, for example adding a canary release to a semantic-release CD pipeline is relatively simple.

## install

```bash
$ go install github.com/jeffrom/tunk/cmd/tunk
```

## usage

```bash
tunk [rc]

A utility for creating Semantic Versioning-compliant tags using a commit
message policy.

FLAGS
      --all                    operate on all scopes
  -b, --branch stringArray     set release branches (default [main,master])
  -n, --dry-run                Don't do destructive operations
  -h, --help                   show help
      --major                  bump major version
      --minor                  bump minor version
      --patch                  bump patch version
      --policies stringArray   declare policies to use (default [conventional-lax,lax])
  -q, --quiet                  print as little as necessary
  -s, --scope string           Operate on a scope
      --scopes stringArray     declare release scopes
      --tag-template string    go text/template for tag
  -v, --verbose                print additional debugging info
  -V, --version                print version and exit
```
