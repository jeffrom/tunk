package config

import "regexp"

type Policy struct {
	Name                  string            `json:"name"`
	SubjectRE             string            `json:"subject_regex"`
	BodyAnnotationStartRE string            `json:"body_annotation_start_regex"`
	BreakingChangeTypes   []string          `json:"breaking_change_annotations"`
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

var builtinPolicies = []Policy{
	{
		Name:                  "conventional-lax",
		SubjectRE:             `^(?P<type>[A-Za-z0-9]+)(?P<scope>\([^\)]+\))?!?:\s+(?P<body>.+)$`,
		BodyAnnotationStartRE: `^(?P<name>[A-Z ]+): `,
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
}

func getBuiltinPolicy(name string) *Policy {
	for _, pol := range builtinPolicies {
		if name == pol.Name {
			return &pol
		}
	}
	return nil
}
