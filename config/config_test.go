package config

import "testing"

func TestConfig(t *testing.T) {
	cfg := New(nil)
	if len(cfg.Policies) != 2 {
		t.Fatalf("expected %d policies, got %d", 2, len(cfg.Policies))
	}
}
