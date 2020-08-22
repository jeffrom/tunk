package config

import (
	"fmt"
	"regexp"

	"github.com/imdario/mergo"
)

type Config struct {
	Verbose        bool       `json:"verbose,omitempty"`
	Dryrun         bool       `json:"dryrun,omitempty"`
	Quiet          bool       `json:"quiet,omitempty"`
	ReleaseScopes  []string   `json:"release_scopes,omitempty"`
	Policies       []string   `json:"policies,omitempty"`
	CustomPolicies []Policy   `json:"custom_policies,omitempty"`
	Term           TerminalIO `json:"-"`
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
		if err := mergo.Merge(&cfg, overrides); err != nil {
			panic(err)
		}
	}
	return cfg
}

func GetDefault() Config {
	return Config{
		Verbose:  true,
		Policies: []string{"conventional-lax", "lax"},
		CustomPolicies: []Policy{
			{
				Name:                  "conventional-lax",
				Branches:              []string{"master"},
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
				Branches:            []string{"master"},
				SubjectRE:           `^(?P<scope>[A-Za-z0-9_-]+): `,
				FallbackReleaseType: "PATCH",
			},
		},
	}
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
	if !c.Verbose {
		return
	}
	c.Printf(msg, args...)
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
	Branches              []string          `json:"branches"`
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
