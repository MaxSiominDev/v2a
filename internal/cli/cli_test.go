package cli

import (
	"strings"
	"testing"
)

// Flag validation errors fire before any external tool is touched

func TestCompressFlagValidation(t *testing.T) {
	root := NewRootCmd()
	root.SetArgs([]string{"compress", "song.mp3", "--size", "1mb", "--bitrate", "64k"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("both flags error = %v, want mutual exclusion", err)
	}

	root = NewRootCmd()
	root.SetArgs([]string{"compress", "song.mp3"})
	err = root.Execute()
	if err == nil || !strings.Contains(err.Error(), "--size or --bitrate") {
		t.Errorf("no flags error = %v, want required-mode message", err)
	}

	root = NewRootCmd()
	root.SetArgs([]string{"compress", "song.mp3", "--size", "huge"})
	err = root.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid size") {
		t.Errorf("bad size error = %v, want invalid size", err)
	}

	root = NewRootCmd()
	root.SetArgs([]string{"compress"})
	err = root.Execute()
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("missing arg error = %v, want arg-count error", err)
	}
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1_000, "1 kB"},
		{1_500, "1.5 kB"},
		{48_372, "48.4 kB"},
		{80_000_000, "80 MB"},
		{1_500_000_000, "1.5 GB"},
	}

	for _, tt := range tests {
		if got := humanSize(tt.bytes); got != tt.want {
			t.Errorf("humanSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestFrameRate(t *testing.T) {
	tests := []struct {
		ratio, want string
	}{
		{"30000/1001", "29.97"},
		{"25/1", "25.00"},
		{"25", "25"},
		{"0/0", ""},
		{"x/y", ""},
	}

	for _, tt := range tests {
		if got := frameRate(tt.ratio); got != tt.want {
			t.Errorf("frameRate(%q) = %q, want %q", tt.ratio, got, tt.want)
		}
	}
}
