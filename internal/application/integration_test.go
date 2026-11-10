//go:build integration

package application

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/MaxSiominDev/v2a/internal/compress"
)

// Real ffmpeg/ffprobe required; run with: go test -tags integration ./...
// Media is generated with lavfi, no fixtures are checked in

func requireFFmpeg(t *testing.T) *App {
	t.Helper()

	app, err := New()
	if err != nil {
		t.Skipf("ffmpeg/ffprobe not available: %v", err)
	}
	return app
}

func generate(t *testing.T, dir, name string, args ...string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	full := append([]string{"-v", "error", "-y"}, args...)
	full = append(full, path)

	out, err := exec.Command("ffmpeg", full...).CombinedOutput()
	if err != nil {
		t.Fatalf("failed to generate %s: %v\n%s", name, err, out)
	}
	return path
}

func generateMov(t *testing.T, dir string) string {
	t.Helper()
	return generate(t, dir, "movie.mov",
		"-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=10",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac",
		"-metadata", "title=Integration Movie",
	)
}

func generateMp3(t *testing.T, dir string) string {
	t.Helper()
	return generate(t, dir, "song.mp3",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=10",
		"-c:a", "libmp3lame", "-b:a", "128k",
		"-metadata", "title=Integration Song",
	)
}

func TestIntegrationExtract(t *testing.T) {
	app := requireFFmpeg(t)
	dir := t.TempDir()

	input := generateMov(t, dir)
	res, err := app.Extract(context.Background(), ExtractRequest{Input: input})
	if err != nil {
		t.Fatalf("Extract unexpected error: %v", err)
	}

	probe, err := app.Info(context.Background(), res.OutputPath)
	if err != nil {
		t.Fatalf("Info on output failed: %v", err)
	}
	audio := probe.AudioStreams()
	if len(audio) != 1 || audio[0].CodecName != "mp3" {
		t.Fatalf("output audio = %+v, want one mp3 stream", audio)
	}
	if !probe.HasGlobalTitle() || probe.Metadata()["title"] != "Integration Movie" {
		t.Errorf("title not preserved: %v", probe.Metadata())
	}
}

func TestIntegrationCompressToFit(t *testing.T) {
	app := requireFFmpeg(t)
	dir := t.TempDir()

	input := generateMp3(t, dir)
	res, err := app.Compress(context.Background(), CompressRequest{Input: input, TargetBytes: 100_000})
	if err != nil {
		t.Fatalf("Compress unexpected error: %v", err)
	}

	info, err := os.Stat(res.OutputPath)
	if err != nil {
		t.Fatalf("output missing: %v", err)
	}
	if info.Size() > 100_000 {
		t.Errorf("output size = %d, exceeds target 100000", info.Size())
	}

	probe, err := app.Info(context.Background(), res.OutputPath)
	if err != nil {
		t.Fatalf("Info on output failed: %v", err)
	}
	if !probe.HasGlobalTitle() || probe.Metadata()["title"] != "Integration Song" {
		t.Errorf("title not preserved: %v", probe.Metadata())
	}
}

func TestIntegrationCompressBitrate(t *testing.T) {
	app := requireFFmpeg(t)
	dir := t.TempDir()

	input := generateMp3(t, dir)
	res, err := app.Compress(context.Background(), CompressRequest{Input: input, Bitrate: 64_000})
	if err != nil {
		t.Fatalf("Compress unexpected error: %v", err)
	}

	probe, err := app.Info(context.Background(), res.OutputPath)
	if err != nil {
		t.Fatalf("Info on output failed: %v", err)
	}
	audio := probe.AudioStreams()
	if len(audio) != 1 {
		t.Fatalf("output audio streams = %d, want 1", len(audio))
	}
	if got, _ := strconv.Atoi(audio[0].BitRate); got != 64_000 {
		t.Errorf("output bitrate = %d, want 64000", got)
	}
}

func TestIntegrationExtractBitrateBelowFloor(t *testing.T) {
	app := requireFFmpeg(t)
	dir := t.TempDir()

	input := generateMov(t, dir)
	_, err := app.Extract(context.Background(), ExtractRequest{Input: input, Bitrate: 8_000})
	if !errors.Is(err, compress.ErrBitrateTooLow) {
		t.Fatalf("Extract error = %v, want ErrBitrateTooLow", err)
	}
}

func TestIntegrationExtractCoverArt(t *testing.T) {
	app := requireFFmpeg(t)
	dir := t.TempDir()

	ppm := filepath.Join(dir, "cover.ppm")
	if err := os.WriteFile(ppm, []byte("P1 1 1 0"), 0o644); err != nil {
		t.Fatalf("failed to write cover: %v", err)
	}

	// The MOV muxer drops attached pictures, so the fixture is built as M4A
	// and renamed; v2a only checks the extension
	m4a := generate(t, dir, "covered.m4a",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=2",
		"-i", ppm,
		"-map", "0:a", "-map", "1:v",
		"-c:a", "aac", "-c:v", "mjpeg", "-disposition:v", "attached_pic",
	)
	input := filepath.Join(dir, "covered.mov")
	if err := os.Rename(m4a, input); err != nil {
		t.Fatalf("failed to rename fixture: %v", err)
	}

	res, err := app.Extract(context.Background(), ExtractRequest{Input: input})
	if err != nil {
		t.Fatalf("Extract unexpected error: %v", err)
	}

	probe, err := app.Info(context.Background(), res.OutputPath)
	if err != nil {
		t.Fatalf("Info on output failed: %v", err)
	}
	if probe.AttachedPicture() == nil {
		t.Error("cover art was not carried into the MP3")
	}
}
