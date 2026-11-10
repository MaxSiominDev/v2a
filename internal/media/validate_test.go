package media

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckInputFile(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "movie.mov")
	touchFile(t, existing)

	if err := CheckInputFile(existing); err != nil {
		t.Errorf("CheckInputFile(%q) unexpected error: %v", existing, err)
	}
	if err := CheckInputFile(filepath.Join(dir, "missing.mov")); !errors.Is(err, ErrInputNotFound) {
		t.Errorf("CheckInputFile missing error = %v, want ErrInputNotFound", err)
	}
	err := CheckInputFile(dir)
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("CheckInputFile directory error = %v, want a not-a-regular-file error", err)
	}
}

func TestCheckExtension(t *testing.T) {
	if err := CheckExtension("movie.mov", ".mov"); err != nil {
		t.Errorf("lowercase .mov rejected: %v", err)
	}
	if err := CheckExtension("movie.MOV", ".mov"); err != nil {
		t.Errorf("uppercase .MOV rejected: %v", err)
	}
	if err := CheckExtension("movie.Mov", ".mov"); err != nil {
		t.Errorf("mixed-case .Mov rejected: %v", err)
	}
	if err := CheckExtension("movie.mkv", ".mov"); !errors.Is(err, ErrUnsupportedFormat) {
		t.Errorf(".mkv error = %v, want ErrUnsupportedFormat", err)
	}
	if err := CheckExtension("noext", ".mov"); !errors.Is(err, ErrUnsupportedFormat) {
		t.Errorf("missing extension error = %v, want ErrUnsupportedFormat", err)
	}
}

func touchFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("failed to create %s: %v", path, err)
	}
}
