package commit

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/blang/semver"

	"github.com/jeffrom/tunk/config"
	"github.com/jeffrom/tunk/model"
	"github.com/jeffrom/tunk/vcs"
)

var ErrNoPolicy = errors.New("commit: no policy matched")

type Analyzer struct {
	cfg config.Config
	vcs vcs.Interface
}

func NewAnalyzer(cfg config.Config, vcs vcs.Interface) *Analyzer {
	return &Analyzer{
		cfg: cfg,
		vcs: vcs,
	}
}

func (a *Analyzer) Analyze(ctx context.Context, rc string) ([]*Version, error) {
	var versions []*Version

	// TODO in CI, fetch the main branch. locally, don't fetch.
	mainBranch, err := a.vcs.GetMainBranch(ctx, a.cfg.GetBranches())
	if err != nil {
		return nil, err
	}
	a.cfg.Debugf("main branch is: %q", mainBranch)

	if !a.cfg.IgnorePolicies {
		if err := a.checkPolicies(ctx, mainBranch); err != nil {
			return nil, err
		}
	}

	if a.cfg.Scope == "" {
		rootVersion, err := a.AnalyzeScope(ctx, "", rc)
		if err != nil {
			return nil, err
		}
		if rootVersion != nil {
			versions = append(versions, rootVersion)
		}
	}

	if a.cfg.All {
		for _, scope := range a.cfg.ReleaseScopes {
			ver, err := a.AnalyzeScope(ctx, scope, rc)
			if err != nil {
				return nil, err
			}
			if ver != nil {
				versions = append(versions, ver)
			}
		}
	} else if a.cfg.Scope != "" {
		ver, err := a.AnalyzeScope(ctx, a.cfg.Scope, rc)
		if err != nil {
			return nil, err
		}
		if ver != nil {
			versions = append(versions, ver)
		}
	}

	if len(versions) == 0 && !a.cfg.InCI {
		return versions, errors.New("no releaseable commits have been created since the last release was tagged")
	}
	return versions, nil
}

func (a *Analyzer) AnalyzeScope(ctx context.Context, scope, rc string) (*Version, error) {
	glob := buildTagQuery(scope)
	tags, err := a.vcs.ReadTags(ctx, glob)
	if err != nil {
		return nil, err
	}
	// fmt.Println("the tags:", tags)
	latest, err := semverLatest(tags, scope, "")
	if err != nil {
		if errors.Is(err, ErrNoTags) {
			initialTag := "v0.1.0"
			if scope != "" {
				initialTag = scope + "/v0.1.0"
			}
			a.cfg.Warning(`no release tags found. To create one: git tag -a %s -m "initial tag"`, initialTag)
		}
		return nil, err
	}
	// fmt.Println("latest:", latest)

	// handle overrides
	// TODO do this later so we can provide more context in the return struct
	if a.cfg.Major {
		nextVer := latest
		nextVer.Major++
		nextVer.Minor = 0
		nextVer.Patch = 0
		return &Version{Version: nextVer}, nil
	}
	if a.cfg.Minor {
		nextVer := latest
		nextVer.Minor++
		nextVer.Patch = 0
		return &Version{Version: nextVer}, nil
	}
	if a.cfg.Patch {
		nextVer := latest
		nextVer.Patch++
		return &Version{Version: nextVer}, nil
	}

	// fmt.Println("current latest version is:", latestVer)
	// fmt.Println("version to start analysis from:", latest)

	logQuery := fmt.Sprintf("%s..HEAD", buildGitTag(latest, scope, ""))
	a.cfg.Debugf("log: %q", logQuery)
	commits, err := a.vcs.ReadCommits(ctx, logQuery)
	if err != nil {
		return nil, err
	}
	ver, err := a.processCommits(latest, commits, scope, a.cfg.ReleaseScopes)
	if err != nil {
		return nil, err
	}

	if ver != nil && rc != "" {
		tagPrefix := buildTagPrefix(scope)
		tagQuery := fmt.Sprintf("%s%s-%s.*", tagPrefix, ver.Version, rc)
		tags, err := a.vcs.ReadTags(ctx, tagQuery)
		if err != nil && !errors.Is(err, ErrNoTags) {
			return nil, err
		}
		ver.RC = a.buildLatestRCTag(scope, rc, tags)
	}
	return ver, nil
}

func (a *Analyzer) checkPolicies(ctx context.Context, mainBranch string) error {
	// in local env, make sure the current commit is on one of the allowed branches
	// if a.cfg.InCI {
	// 	// XXX
	// }

	currCommit, err := a.vcs.CurrentCommit(ctx)
	if err != nil {
		return err
	}

	ok, err := a.vcs.BranchContains(ctx, currCommit, mainBranch)
	if err != nil {
		return err
	}
	if !ok && !a.cfg.Dryrun {
		return fmt.Errorf("current commit %s is not on the main branch %q", currCommit[:8], mainBranch)
	}
	return nil
}

func (a *Analyzer) processCommits(latest semver.Version, commits []*model.Commit, scope string, allScopes []string) (*Version, error) {
	if len(commits) == 0 {
		return nil, nil
	}

	var acs []*AnalyzedCommit
	var maxCommit *AnalyzedCommit
	var latestCommit *AnalyzedCommit
	for _, commit := range commits {
		a.cfg.Debugf("%s (%s) -> %s", commit.ID[:8], commit.Author, commit.Subject)
		ac, err := a.processCommit(commit)
		if err != nil {
			return nil, err
		}
		// fmt.Println("sup", ac.scope, scope, ac.isScoped(scope, allScopes), allScopes)
		if !ac.isScoped(scope, allScopes) {
			a.cfg.Debugf("skipping out of scope commit %s (scope: %q, commit scope: %q)", commit.ShortID(), scope, ac.Scope)
			continue
		}

		if maxCommit == nil {
			maxCommit = ac
		} else if ac.ReleaseType > maxCommit.ReleaseType {
			maxCommit = ac
		}
		if latestCommit == nil || ac.Commit.CommitterDate.After(latestCommit.Commit.CommitterDate) {
			latestCommit = ac
		}

		acs = append(acs, ac)
	}

	if len(acs) == 0 {
		return nil, nil
	}

	a.cfg.Debugf("analyzed: max: %s %s(%q) latest: %s\n", maxCommit.Commit.ShortID(), maxCommit.ReleaseType, maxCommit.Scope, latestCommit.Commit.ShortID())
	if maxCommit.ReleaseType >= ReleasePatch {
		a.cfg.Debugf("%s: will bump %s version (scope: %q)", latestCommit.Commit.ShortID(), maxCommit.ReleaseType, scope)
		nextVersion := bumpVersion(latest, maxCommit.ReleaseType)

		v := &Version{
			Commit:     latestCommit.Commit.ID,
			Version:    nextVersion,
			Scope:      scope,
			AllCommits: acs,
		}
		return v, nil
	}
	return nil, nil
}

func (a *Analyzer) processCommit(commit *model.Commit) (*AnalyzedCommit, error) {
	for _, pol := range a.cfg.GetPolicies() {
		subjectRE := pol.GetSubjectRE()
		var subjectMatch []string
		if subjectRE != nil {
			subjectMatch = subjectRE.FindStringSubmatch(commit.Subject)
			a.cfg.Debugf("%s: policy %q subject match: %+v", commit.ShortID(), pol.Name, subjectMatch)
		}

		typeMatch := false
		if len(subjectMatch) > 0 {
			ac := &AnalyzedCommit{Commit: commit, Policy: pol, Valid: true}
			for i, subexp := range subjectRE.SubexpNames() {
				group := subjectMatch[i]
				switch subexp {
				case "type":
					a.cfg.Debugf("%s: policy %q subject type: %q", commit.ShortID(), pol.Name, group)
					commitType := group
					if pol.CommitTypes != nil {
						rt, ok := pol.CommitTypes[commitType]
						if ok {
							ac.ReleaseType = ReleaseTypeFromString(rt)
							typeMatch = true
						}
					}
				case "scope":
					a.cfg.Debugf("%s: policy %q subject scope: %q", commit.ShortID(), pol.Name, group)
					ac.Scope = strings.Trim(group, "~!@#$%^&*()_+`-=[]\\{}|';:\",./<>?")
				}
			}

			if ac.Scope != "" && ac.ReleaseType == 0 && pol.FallbackReleaseType != "" {
				ac.ReleaseType = ReleaseTypeFromString(pol.FallbackReleaseType)
				typeMatch = true
			}

			if typeMatch {
				breaking, err := a.detectBreakingChange(pol, ac)
				if err != nil {
					return nil, err
				}
				if breaking {
					ac.ReleaseType = ReleaseMajor
				}

				a.cfg.Debugf("policy match: %q (%s)", pol.Name, ac.ReleaseType)
				return ac, nil
			}
		}

		if !typeMatch && pol.FallbackReleaseType != "" {
			ac := &AnalyzedCommit{Commit: commit, Policy: pol, Valid: false, ReleaseType: ReleaseTypeFromString(pol.FallbackReleaseType)}
			a.cfg.Debugf("policy fallback: %q (%s)", pol.Name, ac.ReleaseType)
			return ac, nil
		}
	}
	return nil, ErrNoPolicy
}

func (a *Analyzer) detectBreakingChange(pol *config.Policy, ac *AnalyzedCommit) (bool, error) {
	if ac.Annotations == nil {
		annotations, err := a.getBodyAnnotations(pol, ac.Commit.Body)
		if err != nil {
			return false, err
		}
		ac.Annotations = annotations
	}
	for _, annotation := range ac.Annotations {
		for _, bcn := range pol.BreakingChangeTypes {
			if annotation.Name == bcn {
				return true, nil
			}
		}
	}
	return false, nil
}

func (a *Analyzer) getBodyAnnotations(pol *config.Policy, body string) ([]BodyAnnotation, error) {
	bodyRE := pol.GetBodyAnnotationRE()
	if bodyRE == nil {
		return nil, nil
	}

	var inAnnotation bool
	var curr strings.Builder
	var currName string
	var annotations []BodyAnnotation
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		match := bodyRE.FindStringSubmatch(line)
		if match == nil {
			if !inAnnotation {
				continue
			}

			curr.WriteString(line)
			curr.WriteString("\n")
			continue
		} else {
			body := strings.TrimRight(curr.String(), "\n")
			if body != "" {
				annotations = append(annotations, BodyAnnotation{Name: currName, Body: body})
			}
			curr = strings.Builder{}
			currName = ""
			curr.WriteString(strings.TrimPrefix(line, match[0]))
		}

		inAnnotation = true
		for i, subexp := range bodyRE.SubexpNames() {
			group := match[i]
			switch subexp {
			case "type":
				currName = group
			}
		}

		curr = strings.Builder{}
		curr.WriteString(strings.TrimPrefix(line, match[0]))
		curr.WriteString("\n")
	}

	if body := curr.String(); body != "" && currName != "" {
		annotations = append(annotations, BodyAnnotation{Name: currName, Body: strings.TrimRight(curr.String(), "\n")})
	}
	return annotations, nil
}

func (a *Analyzer) buildLatestRCTag(scope, rc string, tags []string) string {
	prefix := buildTagPrefix(scope)
	var nums []int
	for _, t := range tags {
		trimmed := strings.TrimPrefix(t, prefix)
		parsed, err := semver.Parse(trimmed)
		if err != nil || len(parsed.Pre) != 2 || parsed.Pre[0].String() != rc {
			a.cfg.Warning("invalid tag, skipping: %q (err: %v)", t, err)
			continue
		}

		s := parsed.Pre[1].String()
		n, err := strconv.ParseInt(s, 10, 16)
		if err != nil {
			a.cfg.Warning("invalid tag, skipping: %q, %v", t, err)
			continue
		}
		nums = append(nums, int(n))
	}
	sort.Ints(nums)

	max := 0
	if len(nums) > 0 {
		max = nums[len(nums)-1] + 1
	}
	return fmt.Sprintf("%s.%d", rc, max)
}

func bumpVersion(curr semver.Version, releaseType ReleaseType) semver.Version {
	nextVersion := curr
	switch releaseType {
	case ReleaseMajor:
		nextVersion.Major++
		nextVersion.Minor = 0
		nextVersion.Patch = 0
	case ReleaseMinor:
		nextVersion.Minor++
		nextVersion.Patch = 0
	case ReleasePatch:
		nextVersion.Patch++
	}

	return nextVersion
}

func buildTagQuery(scope string) string {
	if scope == "" {
		return "v*"
	}
	return scope + "/v*"
}

func buildTagPrefix(scope string) string {
	if scope == "" {
		return "v"
	}
	return scope + "/v"
}

type AnalyzedCommit struct {
	*model.Commit
	ReleaseType ReleaseType
	Scope       string
	Policy      *config.Policy
	Valid       bool
	Annotations []BodyAnnotation
}

func (ac *AnalyzedCommit) isScoped(scope string, allScopes []string) bool {
	if ac.Scope == "" {
		for _, other := range allScopes {
			if scope == other {
				return false
			}
		}
		return true
	}
	if len(allScopes) > 0 {
		return scope == ac.Scope
	}
	return true
}

type BodyAnnotation struct {
	Name string
	Body string
}
