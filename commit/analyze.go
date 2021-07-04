package commit

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
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
	tag *Tag
}

func NewAnalyzer(cfg config.Config, vcs vcs.Interface, tag *Tag) *Analyzer {
	if tag == nil {
		var err error
		tag, err = NewTag("")
		if err != nil {
			panic(err)
		}
	}
	return &Analyzer{
		cfg: cfg,
		vcs: vcs,
		tag: tag,
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

func (a *Analyzer) LatestRelease(ctx context.Context, scope, rc string) (semver.Version, error) {
	glob, err := a.tag.Glob(scope, rc)
	if err != nil {
		return semver.Version{}, err
	}
	// fmt.Printf("da glob: %q\n", glob)
	tags, err := a.vcs.ReadTags(ctx, glob)
	if err != nil {
		return semver.Version{}, err
	}
	// fmt.Println("the tags:", tags)
	latest, err := a.tag.SemverLatest(tags, scope, rc)
	if err != nil {
		return semver.Version{}, err
	}
	return latest, nil
}

func (a *Analyzer) ReadCommitsSince(ctx context.Context, scope string, latest semver.Version) ([]*model.Commit, error) {
	q, err := a.tag.ExecuteString(TagData{Version: &Version{Version: latest, Scope: scope}})
	if err != nil {
		return nil, err
	}
	logQuery := fmt.Sprintf("%s..HEAD", q)
	a.cfg.Debugf("log: %q", logQuery)
	return a.vcs.ReadCommits(ctx, logQuery)
}

func (a *Analyzer) AnalyzeScope(ctx context.Context, scope, rc string) (*Version, error) {
	latest, err := a.LatestRelease(ctx, scope, "")
	if err != nil {
		if errors.Is(err, ErrNoTags) {
			initialTag, err := a.tag.ExecuteString(TagData{Version: &Version{Version: semver.Version{Minor: 1}}})
			if err != nil {
				return nil, err
			}
			a.cfg.Warning(`No release tags found. To create one: git tag -a %s -m "initial tag"`, initialTag)
		}
		return nil, err
	}
	// fmt.Println("latest:", latest)

	// fmt.Println("current latest version is:", latestVer)
	// fmt.Println("version to start analysis from:", latest)

	commits, err := a.ReadCommitsSince(ctx, scope, latest)
	if err != nil {
		return nil, err
	}
	ver, err := a.processCommits(latest, commits, scope, a.cfg.ReleaseScopes)
	if err != nil {
		return nil, err
	}

	if ver != nil && rc != "" {
		tagQuery, err := a.tag.GlobVersion(scope, rc, ver.Version)
		if err != nil {
			return nil, err
		}
		// fmt.Printf("glob tag query: %q\n", tagQuery)
		tags, err := a.vcs.ReadTags(ctx, tagQuery)
		if err != nil && !errors.Is(err, ErrNoTags) {
			return nil, err
		}
		pre, err := a.buildLatestRCTag(scope, rc, tags)
		if err != nil {
			return nil, err
		}
		ver.Version.Pre = pre
		// fmt.Printf("should be %q, got %q\n", a.buildLatestRCTag(scope, rc, tags), pre)
	}

	// handle overrides
	if a.cfg.Major {
		nextVer := latest
		nextVer.Pre = ver.Version.Pre
		nextVer.Major++
		nextVer.Minor = 0
		nextVer.Patch = 0
		ver.Version = nextVer
		return ver, nil
	}
	if a.cfg.Minor {
		nextVer := latest
		nextVer.Pre = ver.Version.Pre
		nextVer.Minor++
		nextVer.Patch = 0
		ver.Version = nextVer
		return ver, nil
	}
	if a.cfg.Patch {
		nextVer := latest
		nextVer.Pre = ver.Version.Pre
		nextVer.Patch++
		ver.Version = nextVer
		return ver, nil
	}

	return ver, nil
}

func (a *Analyzer) checkPolicies(ctx context.Context, mainBranch string) error {
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
		ac, err := a.processCommit(commit, a.cfg.GetPolicies())
		if err != nil {
			if errors.Is(err, ErrNoPolicy) && a.cfg.OverridesSet() {
				ac = &AnalyzedCommit{Commit: commit}
			} else {
				return nil, err
			}
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
	} else if a.cfg.OverridesSet() {
		relType := ReleasePatch
		if a.cfg.Minor {
			relType = ReleaseMinor
		} else if a.cfg.Major {
			relType = ReleaseMajor
		}
		return &Version{
			Commit:     latestCommit.Commit.ID,
			Version:    bumpVersion(latest, relType),
			Scope:      scope,
			AllCommits: acs,
		}, nil
	}
	return nil, nil
}

func (a *Analyzer) Match(commit *model.Commit, policies []*config.Policy) (*AnalyzedCommit, error) {
	return a.processCommit(commit, policies)
}

func (a *Analyzer) processCommit(commit *model.Commit, policies []*config.Policy) (*AnalyzedCommit, error) {
	for _, pol := range policies {
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
					ac.CommitType = commitType
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

func (a *Analyzer) buildLatestRCTag(scope, rc string, tags []string) ([]semver.PRVersion, error) {
	var nums []int
	for _, t := range tags {
		parsed, err := a.tag.ExtractSemver(scope, rc, t)
		if err != nil {
			a.cfg.Warning("invalid tag, skipping: %q (err: %v)", t, err)
			continue
		} else if len(parsed.Pre) != 2 {
			a.cfg.Warning("invalid tag, skipping: %q", t)
			continue
		} else if parsed.Pre[0].String() != rc {
			a.cfg.Warning("tag doesn't match rc %q, skipping: %q", rc, t)
			continue
		}

		if !validTunkPre(parsed.Pre) {
			a.cfg.Warning("invalid tag, skipping: %q, %v", t, err)
			continue
		}

		n := parsed.Pre[1].VersionNum
		nums = append(nums, int(n))
	}
	sort.Ints(nums)

	max := 0
	if len(nums) > 0 {
		max = nums[len(nums)-1] + 1
	}
	return []semver.PRVersion{
		{VersionStr: rc},
		{VersionNum: uint64(max), IsNum: true},
	}, nil
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

type AnalyzedCommit struct {
	*model.Commit
	ReleaseType ReleaseType
	Scope       string
	CommitType  string
	Policy      *config.Policy
	// Valid, when false, indicates that the commit didn't match any policies,
	// but there was a fallback.
	Valid       bool
	Annotations []BodyAnnotation
}

type AnalyzedCommits []*AnalyzedCommit

func (acs AnalyzedCommits) TextSummary(w io.Writer) error {
	bw := bufio.NewWriter(w)

	multi := len(acs) > 1
	if multi {
		bw.WriteString(fmt.Sprintf("%d commits\n", len(acs)))
	}
	for _, ac := range acs {
		if multi {
			bw.WriteString(ac.Commit.Subject)
			bw.WriteString("\n")
		}
		bw.WriteString(fmt.Sprintf("  Release type: %s\n", ac.ReleaseType))
		if ac.Scope != "" {
			bw.WriteString(fmt.Sprintf("  Scope: %s\n", ac.Scope))
		}
		if ac.CommitType != "" {
			bw.WriteString(fmt.Sprintf("  Commit Type: %s\n", ac.CommitType))
		}
	}
	return bw.Flush()
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
