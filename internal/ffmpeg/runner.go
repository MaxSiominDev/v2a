package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type RunError struct {
	Tool   string
	Stderr string
	Err    error
}

func (e *RunError) Error() string {
	return fmt.Sprintf("%s exited with an error: %v", e.Tool, e.Err)
}

func (e *RunError) Unwrap() error { return e.Err }

// Runner runs external tools and returns their stdout and stderr
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout []byte, stderr []byte, err error)
}

type execRunner struct{}

func NewRunner() Runner { return execRunner{} }

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stdout.Bytes(), stderr.Bytes(), &RunError{Tool: name, Stderr: stderr.String(), Err: err}
	}
	return stdout.Bytes(), stderr.Bytes(), nil
}
