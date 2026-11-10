package media

import (
	"context"
	"errors"
	"testing"
)

const probeJSON = `{
  "streams": [
    {"index": 0, "codec_name": "h264", "codec_type": "video", "width": 1920, "height": 1080,
     "avg_frame_rate": "30000/1001",
     "disposition": {"attached_pic": 0}},
    {"index": 1, "codec_name": "aac", "codec_type": "audio", "bit_rate": "128000",
     "sample_rate": "44100", "channels": 2,
     "disposition": {"attached_pic": 0},
     "tags": {"title": "Main Track", "language": "eng"}},
    {"index": 2, "codec_name": "mjpeg", "codec_type": "video", "disposition": {"attached_pic": 1}}
  ],
  "format": {"filename": "movie.mov", "duration": "12.345000", "size": "68062", "bit_rate": "9000000",
   "tags": {"title": "Test Movie", "artist": "Someone", "rotation": 90}}
}`

type fakeRunner struct {
	stdout   string
	stderr   string
	err      error
	lastArgs []string
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	f.lastArgs = args
	if f.err != nil {
		return nil, []byte(f.stderr), f.err
	}
	return []byte(f.stdout), nil, nil
}

func newTestProber(stdout string) *Prober {
	return NewProber(&fakeRunner{stdout: stdout}, "ffprobe")
}

func probeFixture(t *testing.T) *ProbeResult {
	t.Helper()
	p := newTestProber(probeJSON)
	result, err := p.Probe(context.Background(), "movie.mov")
	if err != nil {
		t.Fatalf("Probe unexpected error: %v", err)
	}
	return result
}

func TestProbeStreams(t *testing.T) {
	result := probeFixture(t)

	if got := len(result.AudioStreams()); got != 1 {
		t.Fatalf("AudioStreams length = %d, want 1", got)
	}
	audio := result.AudioStreams()[0]
	if audio.CodecName != "aac" || audio.SampleRate != "44100" || audio.Channels != 2 {
		t.Errorf("audio stream = %+v", audio)
	}
	if audio.BitRate != "128000" {
		t.Errorf("audio BitRate = %q, want 128000", audio.BitRate)
	}
	if title, ok := audio.Title(); !ok || title != "Main Track" {
		t.Errorf("audio Title = %q, %v", title, ok)
	}
	if lang, ok := audio.Language(); !ok || lang != "eng" {
		t.Errorf("audio Language = %q, %v", lang, ok)
	}
}

func TestProbeAttachedPicture(t *testing.T) {
	result := probeFixture(t)

	pic := result.AttachedPicture()
	if pic == nil {
		t.Fatal("AttachedPicture = nil, want the mjpeg stream")
	}
	if pic.Index != 2 {
		t.Errorf("attached picture index = %d, want 2", pic.Index)
	}
}

func TestProbeFormat(t *testing.T) {
	result := probeFixture(t)

	seconds, err := result.DurationSeconds()
	if err != nil || seconds != 12.345 {
		t.Errorf("DurationSeconds = %v, %v", seconds, err)
	}
	size, err := result.SizeBytes()
	if err != nil || size != 68062 {
		t.Errorf("SizeBytes = %d, %v", size, err)
	}
	if !result.HasGlobalTitle() {
		t.Error("HasGlobalTitle = false, want true")
	}
	if artist := result.Metadata()["artist"]; artist != "Someone" {
		t.Errorf("Metadata artist = %q, want Someone", artist)
	}
}

func TestProbeStreamTitleFallbacksToName(t *testing.T) {
	result := probeFixture(t)
	audio := result.AudioStreams()[0]
	audio.Tags = tagMap{"name": TagValue("Stream Title Here"), "handler_name": TagValue("SoundHandler")}

	if title, ok := audio.Title(); !ok || title != "Stream Title Here" {
		t.Errorf("Title = %q, %v, want Stream Title Here", title, ok)
	}
}

func TestTagValueAcceptsNumbers(t *testing.T) {
	result := probeFixture(t)
	if rotation := result.Metadata()["rotation"]; rotation != "90" {
		t.Errorf("numeric tag rotation = %q, want 90", rotation)
	}
}

func TestProbeToolFailure(t *testing.T) {
	runner := &fakeRunner{stderr: "moov atom not found", err: errors.New("exit status 1")}
	p := NewProber(runner, "ffprobe")

	_, err := p.Probe(context.Background(), "broken.mov")
	var probeErr *ProbeError
	if !errors.As(err, &probeErr) {
		t.Fatalf("Probe error = %v, want *ProbeError", err)
	}
	if probeErr.Stderr != "moov atom not found" {
		t.Errorf("ProbeError.Stderr = %q", probeErr.Stderr)
	}
}

func TestProbeBadJSON(t *testing.T) {
	p := newTestProber("not json")
	if _, err := p.Probe(context.Background(), "x.mov"); err == nil {
		t.Fatal("Probe with invalid JSON unexpectedly succeeded")
	}
}
