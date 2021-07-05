tunk-ci(1)

# NAME

tunk-ci - usage information for continuous integration mode

# SYNOPSIS

tunk can be used in continuous integration settings to create and push tags, and
to validate commit messages. There are two ways to run in this mode: either by
using the *--ci* flag, or if the _$CI_ environment variable is set to _true_,
_1_, or _yes_.

Authorization with the git server is accomplished using _GIT_ASKPASS_. SSH
should work as well, as long as its configured correctly. This means:

- the ssh key is passwordless
- the remote host has already been authorized by the ssh client

# BEHAVIOR

The behavior of tunk in CI mode differs from normal mode in a few ways, most
notably that tunk will push tags after creating them. Tags are pushed using *git
push --follow-tags --atomic*.

# ENVIRONMENT VARIABLES

The following environment variables can be used to configure tunk in CI mode:

*CI*
	enables CI mode if set to _true_, _1_, or _yes_

*GIT_TOKEN, GITHUB_TOKEN, GH_TOKEN*
	The git remote password. Read in order. The first set variable will be used.

# SEE ALSO

*tunk*(1), *tunk-config*(5)

# AUTHORS
