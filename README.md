# trunk-release

trunk-release is an automation tool for tagging releases using a trunk-based development workflow. trunk-release is unlike many other [Semantic Versioning](https://semver.org/) release tools in that it is intended for trunk-based application release, as opposed to branch-based library releases.

There are several similar (and great) tools that serve a similar purpose, such as [semantic-release](https://github.com/semantic-release/semantic-release). semantic-release is intended for branch-based development, and has more opinionated policies about releases than trunk-release. Unlike trunk-release, with semantic-release, in order to publish a release candidate, you typically need to manage a separate branch.

There is also [intuit/auto](https://github.com/intuit/auto). The main difference with trunk-release is reduced feature set. trunk-release does little more than read commits and pushes tags.

Because trunk-release is small, it's easy to combine with other release tools, for example adding a canary release to a semantic-release CD pipeline is relatively simple.

## install

```bash
$ go get github.com/jeffrom/trunk-release/cmd/trunk-release
```

## usage

```bash
$ trunk-release --help
trunk-release [rc]

FLAGS
      --all            operate on all scopes
      --debug          print additional debugging info
  -n, --dry-run        Don't do destructive operations
      --force          force destructive operations
  -h, --help           show help
  -q, --quiet          only print errors
  -s, --scope string   Operate on a scope
  -V, --version        print version and exit
```
