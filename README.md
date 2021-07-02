# tunk

tunk is an automation tool for tagging releases using a trunk-based development workflow. tunk is unlike some other [Semantic Versioning](https://semver.org/) release tools in that it is intended for trunk-based application release, as opposed to branch-based library releases.

There are several similar (and great) tools that serve a similar purpose, such as [semantic-release](https://github.com/semantic-release/semantic-release). semantic-release is intended for branch-based development, and has different release policies than tunk. With semantic-release, in order to publish a release candidate, you typically need an additional branch. tunk just creates another tag on the main branch.

## install

```bash
$ go install github.com/jeffrom/tunk/cmd/tunk
```

## usage

On a git repository with no tags, running `tunk` will print a message that no release tags could be found, and steps for creating one. In the default configuration, the initial tag will look like this: `v0.1.0`. Once a tag has been created, subsequent commits can be automatically tagged.

Running `tunk` on a repository with matching tags will open `$EDITOR` (or `$GIT_EDITOR`) with a summary of the commits that comprise the pending release for final edits. Once saved, a tag will be created for the current commit.

### policies

### templates
