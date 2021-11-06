package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jeffrom/tunk/commit"
	"github.com/jeffrom/tunk/config"
	"github.com/jeffrom/tunk/model"
)

type CheckFailure struct {
	Failures []FailureEntry
}

type FailureEntry struct {
	rawLine     string
	commitID    string
	commitTitle string
	err         error
}

type failuresByCommit struct {
	commits []CheckFailure
}

func (cf CheckFailure) Error() string {
	return fmt.Sprintf("%d check(s) failed", len(cf.Failures))
}

func (cf CheckFailure) Is(other error) bool {
	_, ok := other.(CheckFailure)
	return ok
}

func (cf CheckFailure) WriteFailure(w io.Writer) error {
	if len(cf.Failures) == 0 {
		return nil
	}
	bw := bufio.NewWriter(w)

	byCommit := failuresByCommit{}
	for _, failure := range cf.Failures {
		foundPrev := false
		for _, c := range byCommit.commits {
			match := false
			for _, prevFailure := range c.Failures {
				// fmt.Printf("prev %+v, curr %+v\n", prevFailure, failure)
				if failure.commitID != "" && prevFailure.commitID != "" && failure.commitID == prevFailure.commitID {
					match = true
					break
				}
				if failure.commitTitle != "" && prevFailure.commitTitle != "" && failure.commitTitle == prevFailure.commitTitle {
					match = true
					break
				}
			}
			if match {
				foundPrev = true
				c.Failures = append(c.Failures, failure)
			}
		}

		if !foundPrev {
			byCommit.commits = append(byCommit.commits, CheckFailure{Failures: []FailureEntry{failure}})
		}
	}

	for _, c := range byCommit.commits {
		if len(c.Failures) == 0 {
			continue
		}
		bw.WriteString(c.Failures[0].commitTitle)
		bw.WriteString("\n")
		for _, failure := range c.Failures {
			bw.WriteString("  ")
			bw.WriteString(failure.err.Error())
			bw.WriteString("\n")
		}
	}

	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
}

func (r *Runner) CheckCommits(ctx context.Context, commits []string) (commit.AnalyzedCommits, error) {
	var failures []FailureEntry
	policies := r.cfg.GetPolicies()
	var acs commit.AnalyzedCommits
	for _, c := range commits {
		mc, err := r.parseCommit(c)
		if err != nil {
			failures = append(failures, FailureEntry{rawLine: c, err: err})
			continue
		}

		ac, err := r.analyzer.Match(mc, policies)
		if err != nil {
			failures = append(failures, FailureEntry{commitID: mc.ID, commitTitle: mc.Subject, err: err})
			continue
		}
		acs = append(acs, ac)

		failures = append(failures, r.checkCommit(ac, policies)...)
	}
	if len(failures) > 0 {
		return nil, CheckFailure{Failures: failures}
	}
	return acs, nil
}

func (r *Runner) checkCommit(ac *commit.AnalyzedCommit, policies []*config.Policy) []FailureEntry {
	var failures []FailureEntry

	// if !ac.Valid {
	// 	failures = append(failures, FailureEntry{commitID: mc.ID, commitTitle: mc.Subject, err: errors.New("commit was invalid")})
	// 	continue
	// }
	if ac.Scope != "" && len(r.cfg.AllowedScopes) > 0 && !inStrs(ac.Scope, r.cfg.AllowedScopes) {
		failures = append(failures, FailureEntry{commitID: ac.ID, commitTitle: ac.Subject, err: fmt.Errorf("scope %q is disallowed", ac.Scope)})
	}
	if ac.CommitType != "" && len(r.cfg.AllowedTypes) > 0 && !inStrs(ac.CommitType, r.cfg.AllowedTypes) {
		failures = append(failures, FailureEntry{commitID: ac.ID, commitTitle: ac.Subject, err: fmt.Errorf("commit type %q is disallowed", ac.CommitType)})
	}

	return failures
}

// parseCommits reads raw commit messages
func (r *Runner) parseCommit(s string) (*model.Commit, error) {
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return &model.Commit{Subject: s}, nil
	}
	var cleaned []string
	for _, line := range lines[2:] {
		if strings.HasPrefix(line, "#") {
			continue
		}
		cleaned = append(cleaned, line)
	}
	body := strings.Join(cleaned, "\n")
	return &model.Commit{Subject: lines[0], Body: body}, nil
}

// func (r *Runner) parseGitLogOneline(s string) (*model.Commit, error) {
// 	return nil, nil
// }

func (r *Runner) CheckReadCommit(ctx context.Context, rdr io.Reader) (commit.AnalyzedCommits, error) {
	var failures []FailureEntry
	raw, err := io.ReadAll(rdr)
	if err != nil {
		return nil, err
	}
	acs, err := r.CheckCommits(ctx, []string{string(raw)})
	if err != nil {
		cf := CheckFailure{}
		if !errors.As(err, &cf) {
			return nil, err
		}
		failures = append(failures, cf.Failures...)
	}

	if len(failures) > 0 {
		return nil, CheckFailure{Failures: failures}
	}
	return acs, nil
}

// CheckCommitsFromGit checks all commits since the last release.
func (r *Runner) CheckCommitsFromGit(ctx context.Context, scope string) (commit.AnalyzedCommits, error) {
	if err := r.Check(ctx, ""); err != nil && !isWrongBranchError(err) {
		return nil, err
	}
	latest, err := r.analyzer.LatestRelease(ctx, scope, "")
	if err != nil {
		return nil, err
	}
	commits, err := r.analyzer.ReadCommitsSince(ctx, scope, latest)
	if err != nil {
		return nil, err
	}
	policies := r.cfg.GetPolicies()
	var failures []FailureEntry
	var acs commit.AnalyzedCommits
	for _, mc := range commits {
		ac, err := r.analyzer.Match(mc, policies)
		if err != nil {
			failures = append(failures, FailureEntry{commitID: mc.ID, commitTitle: mc.Subject, err: err})
			continue
		}
		fs := r.checkCommit(ac, policies)
		failures = append(failures, fs...)
		acs = append(acs, ac)
	}

	if len(failures) > 0 {
		return nil, CheckFailure{Failures: failures}
	}
	return acs, nil
}

func inStrs(s string, cands []string) bool {
	for _, cand := range cands {
		if s == cand {
			return true
		}
	}
	return false
}
