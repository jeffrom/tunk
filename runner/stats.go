package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

type Stats struct {
	Commits int64
	Counts  map[string][]*statCount
}

func (s *Stats) Add(bucket, name string, n int64) {
	counts := s.Counts[bucket]
	count, found := s.findCount(name, counts)
	if !found {
		counts = append(counts, count)
	}
	count.Add(n)

	s.Counts[bucket] = counts
}

func (s *Stats) findCount(name string, counts []*statCount) (*statCount, bool) {
	for _, c := range counts {
		if c.label == name {
			return c, true
		}
	}
	return &statCount{label: name}, false
}

func (s *Stats) sortedBuckets() []string {
	buckets := make([]string, len(s.Counts))
	i := 0
	for name := range s.Counts {
		buckets[i] = name
		i++
	}
	sort.Strings(buckets)
	return buckets
}

// func (s *Stats) sortedCounts() [][]*statCount {
// 	buckets := s.sortedBuckets()

// 	sorted := make([][]*statCount, len(s.Counts))
// 	for name, cand := range s.Counts {
// 		for i, bucket := range buckets {
// 			if name == bucket {
// 				sorted[i] = cand
// 				break
// 			}
// 		}
// 	}
// 	return sorted
// }

type statCount struct {
	label string
	n     int64
}

func (c *statCount) Add(n int64) {
	c.n += n
}

func (s *Stats) TextSummary(w io.Writer) error {
	bw := bufio.NewWriter(w)
	bw.WriteString(fmt.Sprintf("%d commits\n\n", s.Commits))

	// b, _ := json.MarshalIndent(s.Counts, "", "  ")
	// fmt.Println(string(b))
	buckets := s.sortedBuckets()
	for _, name := range buckets {
		counts := s.Counts[name]
		sort.Slice(counts, func(i, j int) bool {
			return counts[i].n > counts[j].n
		})
		bw.WriteString(fmt.Sprintf("%s:\n", toTitle(name)))
		for _, count := range counts {
			label := count.label
			if label == "" {
				label = "n/a"
			}
			bw.WriteString(fmt.Sprintf("  %20s\t\t%d\n", label, count.n))
		}
		bw.WriteString("\n")
	}
	return bw.Flush()
}

func (r *Runner) Stats(ctx context.Context) (*Stats, error) {
	if err := r.Check(ctx, ""); err != nil && !isWrongBranchError(err) {
		return nil, err
	}

	commits, err := r.vcs.ReadCommits(ctx, r.mainBranch)
	if err != nil {
		return nil, err
	}
	stats := &Stats{
		Commits: int64(len(commits)),
		Counts:  make(map[string][]*statCount),
	}

	policies := r.cfg.GetPolicies()
	for _, c := range commits {
		ac, err := r.analyzer.Match(c, policies)
		if err != nil {
			return nil, err
		}
		stats.Add("scope", ac.Scope, 1)
		stats.Add("commit_type", ac.CommitType, 1)
		stats.Add("type", ac.ReleaseType.String(), 1)
	}
	return stats, nil
}

var nonAlphaRE = regexp.MustCompile(`[^A-Za-z]`)

func toTitle(s string) string {
	s = nonAlphaRE.ReplaceAllLiteralString(s, " ")
	return strings.Title(s)
}
