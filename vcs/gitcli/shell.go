package gitcli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
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

// ArgsString returns a string suitable for copy/paste into the terminal.
func ArgsString(args []string) string {
	b := &bytes.Buffer{}

	for i, arg := range args {
		if strings.Contains(arg, " ") {
			b.WriteString(`"`)
			b.WriteString(arg)
			b.WriteString(`"`)
		} else {
			b.WriteString(arg)
		}

		if i < len(args)-1 {
			b.WriteString(" ")
		}
	}

	return b.String()
}
