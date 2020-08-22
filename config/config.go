package config

import "github.com/imdario/mergo"

type Config struct {
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

type Policy struct {
	Name                  string            `json:"name"`
	Branches              []string          `json:"branches"`
	SubjectRE             string            `json:"subject_regex"`
	BodyAnnotationStartRE string            `json:"body_annotation_start_regex"`
	BreakingChangeTypes   []string          `json:"breaking_change_types"`
	CommitTypes           map[string]string `json:"commit_types"`
	FallbackReleaseType   string            `json:"fallback_type,omitempty"`
}
