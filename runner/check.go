package runner

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"

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

func (r *Runner) CheckCommitSubjects(ctx context.Context, commits []string) error {
	var failures []FailureEntry
	policies := r.cfg.GetPolicies()
	for _, c := range commits {
		mc, err := r.parseCommit(c)
		if err != nil {
			failures = append(failures, FailureEntry{rawLine: c, err: err})
			continue
		}
		failures = append(failures, r.checkCommit(mc, policies)...)
	}
	if len(failures) > 0 {
		return CheckFailure{Failures: failures}
	}
	return nil
}

func (r *Runner) checkCommit(mc *model.Commit, policies []*config.Policy) []FailureEntry {
	var failures []FailureEntry

	ac, err := r.analyzer.Match(mc, policies)
	if err != nil {
		failures = append(failures, FailureEntry{commitID: mc.ID, commitTitle: mc.Subject, err: err})
		return failures
	}
	// if !ac.Valid {
	// 	failures = append(failures, FailureEntry{commitID: mc.ID, commitTitle: mc.Subject, err: errors.New("commit was invalid")})
	// 	continue
	// }
	if ac.Scope != "" && len(r.cfg.AllowedScopes) > 0 && !inStrs(ac.Scope, r.cfg.AllowedScopes) {
		failures = append(failures, FailureEntry{commitID: mc.ID, commitTitle: mc.Subject, err: fmt.Errorf("scope %q is disallowed", ac.Scope)})
	}
	if ac.CommitType != "" && len(r.cfg.AllowedTypes) > 0 && !inStrs(ac.CommitType, r.cfg.AllowedTypes) {
		failures = append(failures, FailureEntry{commitID: mc.ID, commitTitle: mc.Subject, err: fmt.Errorf("commit type %q is disallowed", ac.CommitType)})
	}

	return failures
}

// parseCommits reads raw commit messages
func (r *Runner) parseCommit(s string) (*model.Commit, error) {
	return &model.Commit{Subject: s}, nil
}

// func (r *Runner) parseGitLogOneline(s string) (*model.Commit, error) {
// 	return nil, nil
// }

func (r *Runner) CheckReadCommits(ctx context.Context, rdr io.Reader) error {
	var failures []FailureEntry
	s := bufio.NewScanner(rdr)
	for s.Scan() {
		commit := s.Text()
		if commit == "" {
			continue
		}
		if err := r.CheckCommitSubjects(ctx, []string{commit}); err != nil {
			cf := CheckFailure{}
			if !errors.As(err, &cf) {
				return err
			}
			failures = append(failures, cf.Failures...)
		}
	}
	if err := s.Err(); err != nil {
		return err
	}

	if len(failures) > 0 {
		return CheckFailure{Failures: failures}
	}
	return nil
}

// CheckCommitsFromGit checks all commits since the last release.
func (r *Runner) CheckCommitsFromGit(ctx context.Context) error {
	if err := r.Check(ctx, ""); err != nil && !isWrongBranchError(err) {
		return err
	}
	latest, err := r.analyzer.LatestRelease(ctx, "", "")
	if err != nil {
		return err
	}
	commits, err := r.analyzer.ReadCommitsSince(ctx, "", latest)
	if err != nil {
		return err
	}
	policies := r.cfg.GetPolicies()
	var failures []FailureEntry
	for _, mc := range commits {
		failures = append(failures, r.checkCommit(mc, policies)...)
	}

	if len(failures) > 0 {
		return CheckFailure{Failures: failures}
	}
	return nil
}

func inStrs(s string, cands []string) bool {
	for _, cand := range cands {
		if s == cand {
			return true
		}
	}
	return false
}
