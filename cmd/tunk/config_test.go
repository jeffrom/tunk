package main

import "testing"

func TestInvalidConfig(t *testing.T) {
	tcs := []struct {
		name string
		args []string
	}{
		{
			name: "major-minor",
			args: []string{"--major", "--minor"},
		},
		{
			name: "patch-minor",
			args: []string{"--patch", "--minor"},
		},
		{
			name: "patch-major",
			args: []string{"--patch", "--major"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"tunk", "--dry-run"}, tc.args...)
			t.Logf("args: %q", tc.args)
			if err := run(args); err == nil {
				t.Fatal("expected args to be invalid")
			} else {
				t.Log(err)
			}
		})
	}
}
