// Package config contains tunk's configuration.
package config

import (
	"errors"
	"fmt"

	"github.com/imdario/mergo"
)

type Config struct {
	InCI           bool       `json:"ci,omitempty"`
	Debug          bool       `json:"debug,omitempty"`
	Dryrun         bool       `json:"dryrun,omitempty"`
	Quiet          bool       `json:"quiet,omitempty"`
	All            bool       `json:"all,omitempty"`
	Scope          string     `json:"scope,omitempty"`
	Name           string     `json:"name,omitempty"`
	Major          bool       `json:"major,omitempty"`
	Minor          bool       `json:"minor,omitempty"`
	Patch          bool       `json:"patch,omitempty"`
	Branches       []string   `json:"branches,omitempty"`
	ReleaseScopes  []string   `json:"release_scopes,omitempty"`
	Policies       []string   `json:"policies,omitempty"`
	CustomPolicies []Policy   `json:"custom_policies,omitempty"`
	TagTemplate    string     `json:"tag_template,omitempty"`
	LogTemplate    string     `json:"log_template,omitempty"`
	NoEdit         bool       `json:"no_edit,omitempty"`
	AllowedScopes  []string   `json:"allowed_scopes,omitempty"`
	AllowedTypes   []string   `json:"allowed_types,omitempty"`
	Term           TerminalIO `json:"-"`

	// IgnorePolicies ignores policy restrictions. Intended for testing only.
	IgnorePolicies bool `json:"-"`
	BranchesSet    bool `json:"-"`
}

func New(overrides *Config) Config {
	return NewWithTerminalIO(overrides, nil)
}

func NewWithTerminalIO(overrides *Config, termio *TerminalIO) Config {
	cfg := GetDefault()
	if termio == nil {
		termio = &DefaultTermIO
	}
	cfg.Term = *termio

	if overrides != nil {
		if err := mergo.Merge(overrides, cfg); err != nil {
			panic(err)
		}
		// fmt.Printf("merged: %+v\n", *overrides)
		return *overrides
	}
	return cfg
}

func (c Config) Validate() error {
	if (c.Major && (c.Minor || c.Patch)) ||
		(c.Minor && (c.Patch)) {
		return errors.New("only one of --major, --minor, and --patch is allowed")
	}
	return nil
}

func (c Config) Printf(msg string, args ...interface{}) {
	if c.Quiet {
		return
	}
	fmt.Fprintf(c.Term.Stdout, msg+"\n", args...)
}

func (c Config) Errorf(msg string, args ...interface{}) {
	fmt.Fprintf(c.Term.Stderr, msg+"\n", args...)
}

func (c Config) Debugf(msg string, args ...interface{}) {
	if !c.Debug {
		return
	}
	c.Printf(msg, args...)
}

func (c Config) Warning(msg string, args ...interface{}) {
	c.Errorf("WARNING: "+msg, args...)
}

func (c Config) GetPolicies() []*Policy {
	policies := make(map[string]*Policy)
	for _, name := range c.Policies {
		if customPol := c.getCustomPolicy(name); customPol != nil {
			policies[name] = customPol
		} else if builtinPol := getBuiltinPolicy(name); builtinPol != nil {
			policies[name] = builtinPol
		} else {
			panic(fmt.Sprintf("policy %q was not found", name))
		}
	}

	pols := make([]*Policy, len(c.Policies))
	for i, name := range c.Policies {
		pols[i] = policies[name]
	}

	return pols
}

func (c Config) GetPolicy(name string) *Policy {
	for _, pol := range c.CustomPolicies {
		if pol.Name == name {
			return &pol
		}
	}
	for _, pol := range builtinPolicies {
		if pol.Name == name {
			return &pol
		}
	}
	return nil
}

func (c Config) GetBranches() []string { return c.Branches }

func (c Config) OverridesSet() bool {
	return (c.Major || c.Minor || c.Patch)
}

func (c Config) getCustomPolicy(name string) *Policy {
	for _, pol := range c.CustomPolicies {
		if pol.Name == name {
			return &pol
		}
	}
	return nil
}
