package media

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/MaxSiominDev/v2a/internal/ffmpeg"
)

// ProbeResult is the subset of ffprobe output v2a needs
type ProbeResult struct {
	Streams []Stream `json:"streams"`
	Format  Format   `json:"format"`
}

type Stream struct {
	Index        int         `json:"index"`
	CodecName    string      `json:"codec_name"`
	CodecType    string      `json:"codec_type"`
	BitRate      string      `json:"bit_rate"`
	SampleRate   string      `json:"sample_rate"`
	Channels     int         `json:"channels"`
	Width        int         `json:"width"`
	Height       int         `json:"height"`
	AvgFrameRate string      `json:"avg_frame_rate"`
	Disposition  Disposition `json:"disposition"`
	Tags         tagMap      `json:"tags,omitempty"`
}

type Disposition struct {
	AttachedPic int `json:"attached_pic"`
}

type Format struct {
	Filename string `json:"filename"`
	Duration string `json:"duration"`
	Size     string `json:"size"`
	BitRate  string `json:"bit_rate"`
	Tags     tagMap `json:"tags,omitempty"`
}

func (r *ProbeResult) AudioStreams() []Stream {
	return r.streamsOfType("audio")
}

// AttachedPicture returns the stream holding the embedded cover art, if any.
func (r *ProbeResult) AttachedPicture() *Stream {
	for i := range r.Streams {
		s := &r.Streams[i]
		if s.CodecType == "video" && s.Disposition.AttachedPic == 1 {
			return s
		}
	}
	return nil
}

func (r *ProbeResult) HasGlobalTitle() bool {
	_, ok := r.Format.Tags.lookup("title")
	return ok
}

func (r *ProbeResult) DurationSeconds() (float64, error) {
	return parseSeconds(r.Format.Duration)
}

func (r *ProbeResult) SizeBytes() (int64, error) {
	return strconv.ParseInt(r.Format.Size, 10, 64)
}

func (r *ProbeResult) Metadata() map[string]string {
	return r.Format.Tags.toStrings()
}

func (r *ProbeResult) streamsOfType(codecType string) []Stream {
	var out []Stream
	for _, s := range r.Streams {
		if s.CodecType == codecType {
			out = append(out, s)
		}
	}
	return out
}

func (s *Stream) Language() (string, bool) {
	return s.Tags.lookup("language")
}

// Title checks the "title" tag, then "name": MOV-family containers store
// stream titles under "name".
func (s *Stream) Title() (string, bool) {
	if v, ok := s.Tags.lookup("title"); ok {
		return v, true
	}
	return s.Tags.lookup("name")
}

type ProbeError struct {
	Path   string
	Stderr string
	Cause  error
}

func (e *ProbeError) Error() string {
	return fmt.Sprintf("failed to inspect %s", e.Path)
}

func (e *ProbeError) Unwrap() error { return e.Cause }

type Prober struct {
	runner      ffmpeg.Runner
	ffprobePath string
}

func NewProber(runner ffmpeg.Runner, ffprobePath string) *Prober {
	return &Prober{runner: runner, ffprobePath: ffprobePath}
}

func (p *Prober) Probe(ctx context.Context, path string) (*ProbeResult, error) {
	args := []string{
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-i", path,
	}

	stdout, stderr, err := p.runner.Run(ctx, p.ffprobePath, args...)
	if err != nil {
		return nil, &ProbeError{Path: path, Stderr: string(stderr), Cause: err}
	}

	var result ProbeResult
	if err := json.Unmarshal(stdout, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output for %s: %w", path, err)
	}
	return &result, nil
}

func DurationString(seconds float64) string {
	return time.Duration(seconds * float64(time.Second)).String()
}
