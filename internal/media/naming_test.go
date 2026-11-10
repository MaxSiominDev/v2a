package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithExtension(t *testing.T) {
	tests := []struct {
		path, ext, want string
	}{
		{"movie.mov", ".mp3", "movie.mp3"},
		{"movie.MOV", ".mp3", "movie.mp3"},
		{"/a/b/movie.mov", ".mp3", "/a/b/movie.mp3"},
		{"noext", ".mp3", "noext.mp3"},
	}

	for _, tt := range tests {
		if got := WithExtension(tt.path, tt.ext); got != tt.want {
			t.Errorf("WithExtension(%q, %q) = %q, want %q", tt.path, tt.ext, got, tt.want)
		}
	}
}

func TestSuffixedName(t *testing.T) {
	tests := []struct {
		path, suffix, want string
	}{
		{"song.mp3", "compressed", "song.compressed.mp3"},
		{"/a/b/song.mp3", "compressed", "/a/b/song.compressed.mp3"},
	}

	for _, tt := range tests {
		if got := SuffixedName(tt.path, tt.suffix); got != tt.want {
			t.Errorf("SuffixedName(%q, %q) = %q, want %q", tt.path, tt.suffix, got, tt.want)
		}
	}
}

func TestCollisionPath(t *testing.T) {
	if got := CollisionPath("movie.mp3", 1); got != "movie (1).mp3" {
		t.Errorf("CollisionPath = %q, want %q", got, "movie (1).mp3")
	}
	if got := CollisionPath("song.compressed.mp3", 2); got != "song.compressed (2).mp3" {
		t.Errorf("CollisionPath = %q, want %q", got, "song.compressed (2).mp3")
	}
}

func TestResolveCollision(t *testing.T) {
	dir := t.TempDir()

	// Nothing exists: the path itself wins.
	first := filepath.Join(dir, "movie.mp3")
	got, err := ResolveCollision(first)
	if err != nil {
		t.Fatalf("ResolveCollision unexpected error: %v", err)
	}
	if got != first {
		t.Errorf("ResolveCollision = %q, want %q", got, first)
	}

	// Each occupied variant pushes the next candidate one number up
	touch := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
	}
	touch("movie.mp3")
	touch("movie (1).mp3")
	touch("movie (2).mp3")

	got, err = ResolveCollision(first)
	if err != nil {
		t.Fatalf("ResolveCollision unexpected error: %v", err)
	}
	want := filepath.Join(dir, "movie (3).mp3")
	if got != want {
		t.Errorf("ResolveCollision = %q, want %q", got, want)
	}
}
