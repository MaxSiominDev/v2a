package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/MaxSiominDev/v2a/internal/compress"
	"github.com/MaxSiominDev/v2a/internal/media"
)

var (
	ErrCompressionModeRequired = errors.New("either a target size or a bitrate is required")
	ErrCompressionModeConflict = errors.New("target size and bitrate cannot be combined")
)

type CompressRequest struct {
	Input       string
	Output      string // explicit output path, empty for the default name
	TargetBytes int64  // size mode: hard limit for the final file
	Bitrate     int    // bitrate mode: bits per second to encode at
}

type CompressResult struct {
	OutputPath string
	// AlreadyWithinTarget: the input satisfied a --size request, no re-encode.
	AlreadyWithinTarget bool
	Copied              bool
	Notes               []string
}

func ValidateCompress(req CompressRequest) error {
	if err := media.CheckInputFile(req.Input); err != nil {
		return err
	}
	return media.CheckExtension(req.Input, ".mp3")
}

// Compress re-encodes an MP3 at a requested bitrate or to fit a requested
// maximum file size. The input is never modified.
func (a *App) Compress(ctx context.Context, req CompressRequest) (*CompressResult, error) {
	if req.TargetBytes > 0 && req.Bitrate > 0 {
		return nil, ErrCompressionModeConflict
	}
	if req.TargetBytes == 0 && req.Bitrate == 0 {
		return nil, ErrCompressionModeRequired
	}
	if req.TargetBytes < 0 || req.Bitrate < 0 {
		return nil, errors.New("target size and bitrate must not be negative")
	}

	if err := ValidateCompress(req); err != nil {
		return nil, err
	}

	probe, err := a.probe(ctx, req.Input)
	if err != nil {
		return nil, err
	}
	if err := media.CheckSingleAudioStream(probe); err != nil {
		return nil, err
	}

	var notes []string
	if req.Bitrate > 0 {
		snapped, err := resolveBitrate(req.Bitrate, probe)
		if err != nil {
			return nil, err
		}
		if snapped != req.Bitrate {
			notes = append(notes, fmt.Sprintf("bitrate %s snapped to %s (nearest supported step)", bitrateString(req.Bitrate), bitrateString(snapped)))
		}
		req.Bitrate = snapped

		if err := checkBitrateBelowSource(req.Bitrate, probe); err != nil {
			return nil, err
		}
	}

	if req.TargetBytes > 0 {
		inputSize, err := probe.SizeBytes()
		if err == nil && inputSize <= req.TargetBytes {
			return a.copyWithinTarget(req, inputSize)
		}
	}

	return a.reencode(ctx, req, probe, notes)
}

// Re-encoding at or above the source bitrate only loses quality.
func checkBitrateBelowSource(bitrate int, probe *media.ProbeResult) error {
	source := sourceBitrate(probe)
	if source == 0 {
		return nil
	}
	if bitrate >= source {
		return fmt.Errorf("%w: source is %s", compress.ErrBitrateNotLower, bitrateString(source))
	}
	return nil
}

func (a *App) copyWithinTarget(req CompressRequest, inputSize int64) (*CompressResult, error) {
	if req.Output == "" {
		return &CompressResult{AlreadyWithinTarget: true}, nil
	}

	out, err := reserveOutput(req.Output)
	if err != nil {
		return nil, err
	}

	src, err := os.Open(req.Input)
	if err != nil {
		out.Discard()
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(out.Temp())
	if err != nil {
		out.Discard()
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		out.Discard()
		return nil, fmt.Errorf("failed to copy input: %w", copyErr)
	}
	if closeErr != nil {
		out.Discard()
		return nil, fmt.Errorf("failed to write output: %w", closeErr)
	}

	if err := out.Commit(); err != nil {
		out.Discard()
		return nil, err
	}

	return &CompressResult{
		OutputPath:          out.final,
		AlreadyWithinTarget: true,
		Copied:              true,
		Notes:               []string{fmt.Sprintf("input is already within the target size (%d bytes)", inputSize)},
	}, nil
}

func (a *App) reencode(ctx context.Context, req CompressRequest, probe *media.ProbeResult, notes []string) (*CompressResult, error) {
	final := req.Output
	if final == "" {
		var err error
		final, err = media.ResolveCollision(media.SuffixedName(req.Input, "compressed"))
		if err != nil {
			return nil, err
		}
	}

	out, err := reserveOutput(final)
	if err != nil {
		return nil, err
	}

	bitrate := req.Bitrate

	if req.TargetBytes > 0 {
		found, searchNotes, err := a.searchBitrate(ctx, req, probe, out)
		if err != nil {
			out.Discard()
			return nil, err
		}
		bitrate = found
		notes = append(notes, searchNotes...)
		notes = append(notes, fmt.Sprintf("encoding at %s to fit the target", bitrateString(bitrate)))
		// The temp file already holds the verified, fitting encode.
	} else {
		encodeReq, buildNotes := buildMP3Request(req.Input, out.Temp(), bitrate, probe)
		notes = append(notes, buildNotes...)

		if err := a.encode(ctx, encodeReq); err != nil {
			out.Discard()
			return nil, err
		}
		if statOrZero(out.Temp()) == 0 {
			out.Discard()
			return nil, errors.New("ffmpeg produced no output")
		}
	}

	if req.TargetBytes > 0 && statOrZero(out.Temp()) > req.TargetBytes {
		out.Discard()
		return nil, fmt.Errorf("%w: result of %d bytes exceeds the %d byte target", compress.ErrTargetTooSmall, statOrZero(out.Temp()), req.TargetBytes)
	}

	if err := out.Commit(); err != nil {
		out.Discard()
		return nil, err
	}

	return &CompressResult{OutputPath: final, Notes: notes}, nil
}

// searchBitrate encodes, measures the real output, and corrects until the
// file fits the target. Bounded by MaxAttempts.
func (a *App) searchBitrate(ctx context.Context, req CompressRequest, probe *media.ProbeResult, out *outputFile) (int, []string, error) {
	duration, err := probe.DurationSeconds()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to determine audio duration: %w", err)
	}

	inputSize, err := probe.SizeBytes()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to determine input size: %w", err)
	}

	// First guess of the non-audio overhead: whatever the input spends beyond
	// its own audio payload.
	var overhead int64
	if src := sourceBitrate(probe); src > 0 {
		if est := inputSize - int64(float64(src)*duration/8); est > 0 {
			overhead = est
		}
	}

	bitrate, err := compress.FitBitrate(req.TargetBytes, overhead, duration, sourceSampleRate(probe))
	if err != nil {
		return 0, nil, err
	}

	var notes []string

	for attempt := 1; attempt <= compress.MaxAttempts; attempt++ {
		encodeReq, buildNotes := buildMP3Request(req.Input, out.Temp(), bitrate, probe)
		notes = buildNotes
		if err := a.encode(ctx, encodeReq); err != nil {
			return 0, nil, err
		}

		size := statOrZero(out.Temp())
		if size == 0 {
			return 0, nil, errors.New("ffmpeg produced no output")
		}
		if size <= req.TargetBytes {
			return bitrate, notes, nil
		}

		// This attempt measured the true overhead; correct and retry.
		overhead = size - int64(float64(bitrate)*duration/8)
		bitrate, err = compress.FitBitrate(req.TargetBytes, overhead, duration, sourceSampleRate(probe))
		if err != nil {
			return 0, nil, err
		}
	}

	return 0, nil, fmt.Errorf("%w: could not fit %d bytes within %d attempts", compress.ErrTargetTooSmall, req.TargetBytes, compress.MaxAttempts)
}
