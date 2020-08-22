package config

import (
	"fmt"
	"io"
	"os"
)

type TerminalIO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

var DefaultTermIO = TerminalIO{
	Stdin:  os.Stdin,
	Stdout: os.Stdout,
	Stderr: os.Stderr,
}

func (t *TerminalIO) Printf(msg string, args ...interface{}) {
	fmt.Fprintf(t.Stdout, msg, args...)
}
