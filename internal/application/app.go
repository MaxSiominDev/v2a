package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/MaxSiominDev/v2a/internal/compress"
	"github.com/MaxSiominDev/v2a/internal/ffmpeg"
	"github.com/MaxSiominDev/v2a/internal/media"
)

var ErrEncodingFailed = errors.New("encoding failed")

// EncodeError carries the raw ffmpeg stderr for verbose mode.
type EncodeError struct {
	Cause  error
	Stderr string
}

func (e *EncodeError) Error() string { return ErrEncodingFailed.Error() }

func (e *EncodeError) Unwrap() error { return e.Cause }

// Tags that are MOV/MP4 container internals and meaningless in MP3.
var containerNoiseKeys = map[string]bool{
	"major_brand":       true,
	"minor_version":     true,
	"compatible_brands": true,
}

type App struct {
	runner      ffmpeg.Runner
	ffmpegPath  string
	ffprobePath string
}

func New() (*App, error) {
	locator := ffmpeg.NewToolLocator()
	ffmpegPath, ffprobePath, err := locator.Locate()
	if err != nil {
		return nil, err
	}
	return &App{runner: ffmpeg.NewRunner(), ffmpegPath: ffmpegPath, ffprobePath: ffprobePath}, nil
}

func (a *App) probe(ctx context.Context, path string) (*media.ProbeResult, error) {
	return media.NewProber(a.runner, a.ffprobePath).Probe(ctx, path)
}

func (a *App) encode(ctx context.Context, req ffmpeg.MP3Request) error {
	_, stderr, err := a.runner.Run(ctx, a.ffmpegPath, req.Args()...)
	if err != nil {
		return &EncodeError{Cause: err, Stderr: string(stderr)}
	}
	return nil
}

// resolveBitrate snaps a user bitrate down to a supported step; libmp3lame
// would clamp silently, including up from the floor.
func resolveBitrate(bitrate int, probe *media.ProbeResult) (int, error) {
	snapped := compress.FloorBitrate(bitrate, sourceSampleRate(probe))
	if snapped < compress.MinBitrate {
		return 0, fmt.Errorf("%w: %s", compress.ErrBitrateTooLow, bitrateString(bitrate))
	}
	return snapped, nil
}

// buildMP3Request expects a probe with exactly one audio stream.
func buildMP3Request(input, output string, bitrate int, probe *media.ProbeResult) (ffmpeg.MP3Request, []string) {
	req := ffmpeg.MP3Request{
		Input:             input,
		Output:            output,
		Bitrate:           bitrate,
		CoverIndex:        -1,
		MapGlobalMetadata: true,
	}
	var notes []string

	audio := probe.AudioStreams()[0]

	if pic := probe.AttachedPicture(); pic != nil {
		req.CoverIndex = pic.Index
	}
	if audio.Channels > 2 {
		req.Downmix = true
		notes = append(notes, fmt.Sprintf("audio has %d channels; folding to stereo for MP3", audio.Channels))
	}
	if rate, err := strconv.Atoi(audio.SampleRate); err == nil && rate > 48000 {
		req.Resample = true
		notes = append(notes, fmt.Sprintf("sample rate %d Hz exceeds the MP3 limit; resampling to 48000 Hz", rate))
	}
	if title, ok := audio.Title(); ok && !probe.HasGlobalTitle() {
		req.FallbackTitle = title
	}

	metadata := probe.Metadata()
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if containerNoiseKeys[strings.ToLower(key)] {
			req.ClearMetadataKeys = append(req.ClearMetadataKeys, key)
		}
	}

	return req, notes
}

// sourceBitrate prefers the audio stream's rate and falls back to the
// container rate, which overstates the audio on files with cover art.
func sourceBitrate(probe *media.ProbeResult) int {
	audio := probe.AudioStreams()[0]
	for _, v := range []string{audio.BitRate, probe.Format.BitRate} {
		if bps, err := strconv.Atoi(v); err == nil && bps > 0 {
			return bps
		}
	}
	return 0
}

func sourceSampleRate(probe *media.ProbeResult) int {
	rate, err := strconv.Atoi(probe.AudioStreams()[0].SampleRate)
	if err != nil {
		return 0
	}
	return rate
}

func bitrateString(bps int) string {
	return strconv.Itoa(bps/1000) + "k"
}

func statOrZero(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
