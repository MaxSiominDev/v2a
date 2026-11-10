package application

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MaxSiominDev/v2a/internal/compress"
)

const noTitleProbeJSON = `{
  "streams": [
    {"index": 0, "codec_name": "aac", "codec_type": "audio", "sample_rate": "44100",
     "channels": 2, "disposition": {"attached_pic": 0},
     "tags": {"name": "Stream Title Here", "handler_name": "SoundHandler"}},
    {"index": 1, "codec_name": "mjpeg", "codec_type": "video", "disposition": {"attached_pic": 1}}
  ],
  "format": {"duration": "2.000000", "size": "50000", "bit_rate": "200000",
   "tags": {"major_brand": "qt  ", "compatible_brands": "qt  "}}
}`

func TestBuildMP3RequestFromNoTitleProbe(t *testing.T) {
	app := newTestApp(t, &fakeRunner{probeStdout: noTitleProbeJSON})

	probe, err := app.probe(context.Background(), "movie.mov")
	if err != nil {
		t.Fatalf("probe unexpected error: %v", err)
	}

	req, notes := buildMP3Request("in.mov", "out.mp3", 0, probe)

	if req.FallbackTitle != "Stream Title Here" {
		t.Errorf("FallbackTitle = %q, want the stream name tag", req.FallbackTitle)
	}
	if req.CoverIndex != 1 {
		t.Errorf("CoverIndex = %d, want 1", req.CoverIndex)
	}

	if len(req.ClearMetadataKeys) != 2 {
		t.Errorf("ClearMetadataKeys = %v, want major_brand and compatible_brands", req.ClearMetadataKeys)
	}
	for _, key := range req.ClearMetadataKeys {
		if key != "compatible_brands" && key != "major_brand" {
			t.Errorf("unexpected cleared key %q", key)
		}
	}
	if len(notes) != 0 {
		t.Errorf("notes = %v, want none for a plain stereo file", notes)
	}
}

func TestBuildMP3RequestKeepsGlobalTitleOverStreamTitle(t *testing.T) {
	app := newTestApp(t, &fakeRunner{probeStdout: extractProbeJSON})

	probe, err := app.probe(context.Background(), "movie.mov")
	if err != nil {
		t.Fatalf("probe unexpected error: %v", err)
	}

	req, _ := buildMP3Request("in.mov", "out.mp3", 0, probe)
	if req.FallbackTitle != "" {
		t.Errorf("FallbackTitle = %q, want empty when a global title exists", req.FallbackTitle)
	}
}

func TestExtractBitrateSnapsDown(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "movie.mov", 100)

	runner := &fakeRunner{probeStdout: extractProbeJSON}
	app := newTestApp(t, runner)

	res, err := app.Extract(context.Background(), ExtractRequest{Input: input, Bitrate: 175_000})
	if err != nil {
		t.Fatalf("Extract unexpected error: %v", err)
	}

	want := []int{160_000}
	if len(runner.encBitrates) != 1 || runner.encBitrates[0] != want[0] {
		t.Errorf("encoded bitrate = %v, want [160000]", runner.encBitrates)
	}
	joined := strings.Join(res.Notes, "; ")
	if !strings.Contains(joined, "175k") || !strings.Contains(joined, "160k") {
		t.Errorf("notes %q do not mention the snap", joined)
	}
}

func TestExtractBitrateBelowFloorFails(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "movie.mov", 100)

	app := newTestApp(t, &fakeRunner{probeStdout: extractProbeJSON})

	_, err := app.Extract(context.Background(), ExtractRequest{Input: input, Bitrate: 16_000})
	if !errors.Is(err, compress.ErrBitrateTooLow) {
		t.Fatalf("Extract error = %v, want ErrBitrateTooLow", err)
	}
	if remaining := dirEntries(t, dir); len(remaining) != 1 {
		t.Errorf("leftover files after failure: %v", remaining)
	}
}

func TestCompressExplicitOutputExists(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "song.mp3", 161_222)
	touchInput(t, dir, "taken.mp3", 10)

	app := newTestApp(t, &fakeRunner{probeStdout: compressProbeJSON})

	_, err := app.Compress(context.Background(), CompressRequest{
		Input: input, Output: filepath.Join(dir, "taken.mp3"), Bitrate: 96_000,
	})
	if !errors.Is(err, ErrOutputExists) {
		t.Fatalf("Compress error = %v, want ErrOutputExists", err)
	}
}

func TestCompressBitrateBelowFloorFails(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "song.mp3", 161_222)

	app := newTestApp(t, &fakeRunner{probeStdout: compressProbeJSON})

	_, err := app.Compress(context.Background(), CompressRequest{Input: input, Bitrate: 24_000})
	if !errors.Is(err, compress.ErrBitrateTooLow) {
		t.Fatalf("Compress error = %v, want ErrBitrateTooLow", err)
	}
}
