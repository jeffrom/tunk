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

func GetDefault() Config {
	return Config{
		Policies: []string{"conventional-lax", "lax"},
		Branches: []string{"main", "master"},
		CustomPolicies: []Policy{
			{
				Name:                  "conventional-lax",
				SubjectRE:             `^(?P<type>[A-Za-z0-9]+)(?P<scope>\([^\)]+\))?!?:\s+(?P<body>.+)$`,
				BodyAnnotationStartRE: `^(?P<type>[A-Z ]+): `,
				BreakingChangeTypes:   []string{"BREAKING CHANGE"},
				CommitTypes: map[string]string{
					"feat":        "MINOR",
					"fix":         "PATCH",
					"revert":      "PATCH",
					"cont":        "PATCH",
					"perf":        "PATCH",
					"improvement": "PATCH",
					"refactor":    "PATCH",
					"style":       "PATCH",
					"test":        "SKIP",
					"chore":       "SKIP",
					"docs":        "SKIP",
				},
			},
			{
				Name:                "lax",
				SubjectRE:           `^(?P<scope>[A-Za-z0-9_-]+): `,
				FallbackReleaseType: "PATCH",
			},
		},
	}
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
	var pols []*Policy
	for _, pol := range c.CustomPolicies {
		if !oneOf(pol.Name, c.Policies) {
			continue
		}
		p := pol
		pols = append(pols, &p)
	}
	return pols
}

func (c Config) GetBranches() []string { return c.Branches }

func (c Config) OverridesSet() bool {
	return (c.Major || c.Minor || c.Patch)
}

func oneOf(s string, l []string) bool {
	for _, cand := range l {
		if s == cand {
			return true
		}
	}
	return false
}

type Policy struct {
	Name                  string            `json:"name"`
	SubjectRE             string            `json:"subject_regex"`
	BodyAnnotationStartRE string            `json:"body_annotation_start_regex"`
	BreakingChangeTypes   []string          `json:"breaking_change_types"`
	CommitTypes           map[string]string `json:"commit_types"`
	FallbackReleaseType   string            `json:"fallback_type,omitempty"`
	subjectRE             *regexp.Regexp
	bodyRE                *regexp.Regexp
}

func (p *Policy) GetSubjectRE() *regexp.Regexp {
	if p.SubjectRE == "" {
		return nil
	}
	if p.subjectRE == nil {
		p.subjectRE = regexp.MustCompile(p.SubjectRE)
	}
	return p.subjectRE
}

func (p *Policy) GetBodyAnnotationRE() *regexp.Regexp {
	if p.BodyAnnotationStartRE == "" {
		return nil
	}
	if p.bodyRE == nil {
		p.bodyRE = regexp.MustCompile(p.BodyAnnotationStartRE)
	}
	return p.bodyRE
}
