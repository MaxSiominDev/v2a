package application

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrOutputExists = errors.New("output path already exists")

// outputFile writes atomically: the final name is reserved up front, the
// payload goes into a hidden temp file in the same directory, and the temp is
// renamed over the reservation once verified. Error paths remove both files;
// a crash or kill can leave the hidden temp behind
type outputFile struct {
	final     string
	temp      string
	committed bool
}

func reserveOutput(final string) (*outputFile, error) {
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	reserved, err := os.OpenFile(final, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("%w: %s", ErrOutputExists, final)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	_ = reserved.Close()

	temp, err := os.CreateTemp(filepath.Dir(final), ".v2a-*.mp3")
	if err != nil {
		_ = os.Remove(final)
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	_ = temp.Close()

	return &outputFile{final: final, temp: temp.Name()}, nil
}

func (o *outputFile) Temp() string { return o.temp }

func (o *outputFile) Commit() error {
	if err := os.Rename(o.temp, o.final); err != nil {
		return fmt.Errorf("failed to finalize output file: %w", err)
	}
	o.committed = true
	return nil
}

func (o *outputFile) Discard() {
	_ = os.Remove(o.temp)
	if !o.committed {
		_ = os.Remove(o.final)
	}
}
