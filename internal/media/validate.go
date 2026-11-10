package media

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrInputNotFound        = errors.New("input file does not exist")
	ErrUnsupportedFormat    = errors.New("unsupported input format")
	ErrNoAudioStream        = errors.New("no audio stream found")
	ErrMultipleAudioStreams = errors.New("multiple audio streams found; exactly one is required")
)

// CheckInputFile verifies that path is accessible and a regular file
func CheckInputFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrInputNotFound, path)
		}
		return fmt.Errorf("cannot access input file %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("input is not a regular file: %s", path)
	}
	return nil
}

func CheckExtension(path string, allowed ...string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return fmt.Errorf("%w: %s has no file extension", ErrUnsupportedFormat, path)
	}
	for _, a := range allowed {
		if ext == strings.ToLower(a) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrUnsupportedFormat, ext)
}

func CheckSingleAudioStream(r *ProbeResult) error {
	switch n := len(r.AudioStreams()); n {
	case 0:
		return ErrNoAudioStream
	case 1:
		return nil
	default:
		return ErrMultipleAudioStreams
	}
}
