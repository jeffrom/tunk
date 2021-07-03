package runner

import (
	"bytes"
	"context"
	"testing"

	"github.com/jeffrom/tunk/config"
	"github.com/jeffrom/tunk/vcs/gitcli"
)

// just reading stats on the current dirs repo (probably always tunks own repo)
// for now
func TestStats(t *testing.T) {
	cfg := config.New(nil)
	git := gitcli.New(cfg, "")
	rnr, err := New(cfg, git)
	if err != nil {
		t.Fatal(err)
	}

	stats, err := rnr.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats == nil {
		t.Fatal("expected stats not to be nil")
	}

	b := &bytes.Buffer{}
	if err := stats.TextSummary(b); err != nil {
		t.Fatal(err)
	}
	t.Logf("stats output:\n%s", b.String())

	if stats.Commits == 0 {
		t.Error("expected total commits to be greater than 0")
	}
	if len(stats.Counts) != 3 {
		t.Errorf("expected 3 counters, got %d", len(stats.Counts))
	}

	expectCounters := []string{"scope", "commit_type", "type"}
	for _, expect := range expectCounters {
		counts, ok := stats.Counts[expect]
		if !ok {
			t.Errorf("expected %q counter", expect)
		} else {
			if len(counts) == 0 {
				t.Errorf("expected %q counter not to be empty", expect)
			}
		}
	}
}
