package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/MaxSiominDev/v2a/internal/media"
)

const extractProbeJSON = `{
  "streams": [
    {"index": 0, "codec_name": "h264", "codec_type": "video", "disposition": {"attached_pic": 0}},
    {"index": 1, "codec_name": "aac", "codec_type": "audio", "bit_rate": "128000",
     "sample_rate": "44100", "channels": 2, "disposition": {"attached_pic": 0},
     "tags": {"title": "Main Track"}},
    {"index": 2, "codec_name": "mjpeg", "codec_type": "video", "disposition": {"attached_pic": 1}}
  ],
  "format": {"duration": "10.000000", "size": "68062", "bit_rate": "9000000",
   "tags": {"title": "Test Movie"}}
}`

const compressProbeJSON = `{
  "streams": [
    {"index": 0, "codec_name": "mp3", "codec_type": "audio", "bit_rate": "128000",
     "sample_rate": "44100", "channels": 2, "disposition": {"attached_pic": 0},
     "tags": {"title": "Song"}}
  ],
  "format": {"duration": "10.000000", "size": "161222", "bit_rate": "128000",
   "tags": {"title": "Song"}}
}`

// fakeRunner answers ffprobe with canned JSON and "encodes" by writing
// simulatedOverhead + bitrate*duration/8 + extraBytes bytes.
type fakeRunner struct {
	probeStdout  string
	probeErr     error
	ffmpegErr    error
	ffmpegStderr string
	extraBytes   int64
	emptyOutput  bool

	ffmpegCalls int
	encBitrates []int
	lastArgs    []string
}

const simulatedOverhead = 1_222
const simulatedDuration = 10.0

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	switch name {
	case "ffprobe":
		if f.probeErr != nil {
			return nil, []byte("moov atom not found"), f.probeErr
		}
		return []byte(f.probeStdout), nil, nil
	case "ffmpeg":
		f.ffmpegCalls++
		f.lastArgs = args
		if f.ffmpegErr != nil {
			return nil, []byte(f.ffmpegStderr), f.ffmpegErr
		}

		output := args[len(args)-1]
		if f.emptyOutput {
			return nil, nil, nil
		}

		bitrate := 96_000
		for i, arg := range args {
			if arg == "-b:a" {
				bitrate, _ = strconv.Atoi(args[i+1])
			}
		}
		f.encBitrates = append(f.encBitrates, bitrate)

		size := simulatedOverhead + int64(float64(bitrate)*simulatedDuration/8) + f.extraBytes
		if err := os.WriteFile(output, make([]byte, size), 0o644); err != nil {
			return nil, nil, err
		}
		return nil, nil, nil
	default:
		return nil, nil, errors.New("unexpected tool: " + name)
	}
}

func newTestApp(t *testing.T, runner *fakeRunner) *App {
	t.Helper()
	return &App{runner: runner, ffmpegPath: "ffmpeg", ffprobePath: "ffprobe"}
}

func touchInput(t *testing.T, dir, name string, size int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatalf("failed to create input: %v", err)
	}
	return path
}

func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestExtractHappyPath(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "movie.mov", 100)

	runner := &fakeRunner{probeStdout: extractProbeJSON}
	app := newTestApp(t, runner)

	res, err := app.Extract(context.Background(), ExtractRequest{Input: input})
	if err != nil {
		t.Fatalf("Extract unexpected error: %v", err)
	}

	want := filepath.Join(dir, "movie.mp3")
	if res.OutputPath != want {
		t.Errorf("OutputPath = %q, want %q", res.OutputPath, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("output missing: %v", err)
	}

	if runner.ffmpegCalls != 1 {
		t.Errorf("ffmpeg calls = %d, want 1", runner.ffmpegCalls)
	}
	if got := strings.Join(runner.lastArgs, " "); !strings.Contains(got, "-map 0:2") {
		t.Errorf("expected cover stream mapping, got: %s", got)
	}
}

func TestExtractEncodeFailureLeavesNoOutput(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "movie.mov", 100)

	runner := &fakeRunner{
		probeStdout:  extractProbeJSON,
		ffmpegErr:    errors.New("exit status 1"),
		ffmpegStderr: "Conversion failed!",
	}
	app := newTestApp(t, runner)

	_, err := app.Extract(context.Background(), ExtractRequest{Input: input})
	var enc *EncodeError
	if !errors.As(err, &enc) {
		t.Fatalf("Extract error = %v, want *EncodeError", err)
	}
	if enc.Stderr != "Conversion failed!" {
		t.Errorf("EncodeError.Stderr = %q", enc.Stderr)
	}

	remaining := dirEntries(t, dir)
	if len(remaining) != 1 || remaining[0] != "movie.mov" {
		t.Errorf("leftover files after failure: %v", remaining)
	}
}

func TestExtractEmptyOutputRejected(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "movie.mov", 100)

	runner := &fakeRunner{probeStdout: extractProbeJSON, emptyOutput: true}
	app := newTestApp(t, runner)

	_, err := app.Extract(context.Background(), ExtractRequest{Input: input})
	if err == nil || !strings.Contains(err.Error(), "no output") {
		t.Fatalf("Extract error = %v, want empty-output rejection", err)
	}
	if remaining := dirEntries(t, dir); len(remaining) != 1 {
		t.Errorf("leftover files after failure: %v", remaining)
	}
}

func TestExtractExplicitOutputExists(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "movie.mov", 100)
	touchInput(t, dir, "taken.mp3", 10)

	app := newTestApp(t, &fakeRunner{probeStdout: extractProbeJSON})

	_, err := app.Extract(context.Background(), ExtractRequest{Input: input, Output: filepath.Join(dir, "taken.mp3")})
	if !errors.Is(err, ErrOutputExists) {
		t.Fatalf("Extract error = %v, want ErrOutputExists", err)
	}
}

func TestExtractNestedOutputDirCreated(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "movie.mov", 100)

	app := newTestApp(t, &fakeRunner{probeStdout: extractProbeJSON})

	out := filepath.Join(dir, "a", "b", "movie.mp3")
	res, err := app.Extract(context.Background(), ExtractRequest{Input: input, Output: out})
	if err != nil {
		t.Fatalf("Extract unexpected error: %v", err)
	}
	if res.OutputPath != out {
		t.Errorf("OutputPath = %q, want %q", res.OutputPath, out)
	}
}

func TestExtractUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	input := touchInput(t, dir, "movie.mkv", 100)

	app := newTestApp(t, &fakeRunner{probeStdout: extractProbeJSON})
	_, err := app.Extract(context.Background(), ExtractRequest{Input: input})
	if !errors.Is(err, media.ErrUnsupportedFormat) {
		t.Fatalf("Extract error = %v, want unsupported format", err)
	}
}
