package config

func GetDefault() Config {
	return Config{
		Policies: []string{"conventional-lax", "lax"},
		Branches: []string{"main", "master"},
		CustomPolicies: []Policy{
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
		},
	}
}
