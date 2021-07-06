// Package tunk analyzes commits and creates git tags based on configured
// policies and behavior.
//
// Related packages: config, commit, runner, model, vcs, vcs/gitcli
package tunk

import "github.com/jeffrom/tunk/config"

// Config holds most of the configuration variables for tunk. This struct is
// intended for command-line use, so not all of its attributes are applicable
// to every operation.
//
// See "go doc github.com/jeffrom/tunk/config Config" for more information.
type Config = config.Config
