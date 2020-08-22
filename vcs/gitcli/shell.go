package gitcli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

var CommandContext = exec.CommandContext

func (g *Git) call(ctx context.Context, args []string) ([]byte, error) {
	cmd := CommandContext(ctx, "git", args...)
	cmd.Dir = g.wd

	eb := &bytes.Buffer{}
	ob := &bytes.Buffer{}
	cmd.Stderr = eb
	cmd.Stdout = ob

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("exec: git %q failed: %s (%w)", args, eb.String(), err)
	}
	return ob.Bytes(), err
}
