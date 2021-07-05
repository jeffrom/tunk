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
