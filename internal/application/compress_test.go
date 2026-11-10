package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MaxSiominDev/v2a/internal/compress"
)

// alreadyFitsProbeJSON: same audio in a 500 byte file, so the copy path fires
const alreadyFitsProbeJSON = `{
  "streams": [
    {"index": 0, "codec_name": "mp3", "codec_type": "audio", "bit_rate": "128000",
     "sample_rate": "44100", "channels": 2, "disposition": {"attached_pic": 0},
     "tags": {"title": "Song"}}
  ],
  "format": {"duration": "10.000000", "size": "500", "bit_rate": "128000",
   "tags": {"title": "Song"}}
}`

func TestCompressSizeModeConverges(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "song.mp3", 161_222)

	// The first estimate overshoots the target, so the search corrects once
	runner := &fakeRunner{probeStdout: compressProbeJSON, extraBytes: 500}
	app := newTestApp(t, runner)

	res, err := app.Compress(context.Background(), CompressRequest{Input: input, TargetBytes: 81_500})
	if err != nil {
		t.Fatalf("Compress unexpected error: %v", err)
	}

	if runner.ffmpegCalls != 2 {
		t.Errorf("ffmpeg calls = %d, want 2", runner.ffmpegCalls)
	}

	out, err := os.Stat(res.OutputPath)
	if err != nil {
		t.Fatalf("output missing: %v", err)
	}
	if out.Size() > 81_500 {
		t.Errorf("output size = %d, exceeds target 81500", out.Size())
	}
	want := []int{64_000, 56_000}
	if len(runner.encBitrates) != len(want) || runner.encBitrates[0] != want[0] || runner.encBitrates[1] != want[1] {
		t.Errorf("encoded bitrates = %v, want %v", runner.encBitrates, want)
	}
}

func TestCompressSizeModeTooSmall(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "song.mp3", 161_222)

	app := newTestApp(t, &fakeRunner{probeStdout: compressProbeJSON})

	_, err := app.Compress(context.Background(), CompressRequest{Input: input, TargetBytes: 10_000})
	if !errors.Is(err, compress.ErrTargetTooSmall) {
		t.Fatalf("Compress error = %v, want ErrTargetTooSmall", err)
	}
	if remaining := dirEntries(t, dir); len(remaining) != 1 {
		t.Errorf("leftover files after failure: %v", remaining)
	}
}

func TestCompressAlreadyWithinTargetWithoutOutput(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "song.mp3", 100)

	app := newTestApp(t, &fakeRunner{probeStdout: alreadyFitsProbeJSON})

	res, err := app.Compress(context.Background(), CompressRequest{Input: input, TargetBytes: 1_000})
	if err != nil {
		t.Fatalf("Compress unexpected error: %v", err)
	}
	if !res.AlreadyWithinTarget || res.OutputPath != "" {
		t.Errorf("result = %+v, want no output", res)
	}
	if remaining := dirEntries(t, dir); len(remaining) != 1 {
		t.Errorf("unexpected files created: %v", remaining)
	}
}

func TestCompressAlreadyWithinTargetWithOutput(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "song.mp3", 500)

	app := newTestApp(t, &fakeRunner{probeStdout: alreadyFitsProbeJSON})

	out := filepath.Join(dir, "small.mp3")
	res, err := app.Compress(context.Background(), CompressRequest{Input: input, Output: out, TargetBytes: 1_000})
	if err != nil {
		t.Fatalf("Compress unexpected error: %v", err)
	}
	if !res.AlreadyWithinTarget || !res.Copied || res.OutputPath != out {
		t.Errorf("result = %+v, want a copy at %q", res, out)
	}

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output missing: %v", err)
	}
	if len(got) != 500 {
		t.Errorf("copied size = %d, want 500", len(got))
	}
}

func TestCompressBitrateModeEncodesOnce(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "song.mp3", 161_222)

	runner := &fakeRunner{probeStdout: compressProbeJSON}
	app := newTestApp(t, runner)

	res, err := app.Compress(context.Background(), CompressRequest{Input: input, Bitrate: 96_000})
	if err != nil {
		t.Fatalf("Compress unexpected error: %v", err)
	}
	if runner.ffmpegCalls != 1 {
		t.Errorf("ffmpeg calls = %d, want 1", runner.ffmpegCalls)
	}
	if len(runner.encBitrates) != 1 || runner.encBitrates[0] != 96_000 {
		t.Errorf("encoded bitrates = %v, want [96000]", runner.encBitrates)
	}
	if !strings.HasSuffix(res.OutputPath, "song.compressed.mp3") {
		t.Errorf("OutputPath = %q, want song.compressed.mp3", res.OutputPath)
	}
}

func TestCompressBitrateNotBelowSource(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "song.mp3", 161_222)

	app := newTestApp(t, &fakeRunner{probeStdout: compressProbeJSON})

	for _, bitrate := range []int{128_000, 256_000} {
		_, err := app.Compress(context.Background(), CompressRequest{Input: input, Bitrate: bitrate})
		if !errors.Is(err, compress.ErrBitrateNotLower) {
			t.Errorf("Compress(%d) error = %v, want ErrBitrateNotLower", bitrate, err)
		}
	}
}

func TestCompressWithoutModeFlag(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "song.mp3", 161_222)

	app := newTestApp(t, &fakeRunner{probeStdout: compressProbeJSON})

	_, err := app.Compress(context.Background(), CompressRequest{Input: input})
	if !errors.Is(err, ErrCompressionModeRequired) {
		t.Fatalf("Compress error = %v, want ErrCompressionModeRequired", err)
	}
}

func TestCompressWithConflictingModes(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "song.mp3", 161_222)

	app := newTestApp(t, &fakeRunner{probeStdout: compressProbeJSON})

	_, err := app.Compress(context.Background(), CompressRequest{Input: input, TargetBytes: 1_000_000, Bitrate: 96_000})
	if !errors.Is(err, ErrCompressionModeConflict) {
		t.Fatalf("Compress error = %v, want ErrCompressionModeConflict", err)
	}
}

func TestCompressCollisionNaming(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "song.mp3", 161_222)
	touchInput(t, dir, "song.compressed.mp3", 10)

	runner := &fakeRunner{probeStdout: compressProbeJSON}
	app := newTestApp(t, runner)

	res, err := app.Compress(context.Background(), CompressRequest{Input: input, Bitrate: 96_000})
	if err != nil {
		t.Fatalf("Compress unexpected error: %v", err)
	}
	want := filepath.Join(dir, "song.compressed (1).mp3")
	if res.OutputPath != want {
		t.Errorf("OutputPath = %q, want %q", res.OutputPath, want)
	}
}
