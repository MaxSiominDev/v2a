package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/MaxSiominDev/v2a/internal/media"
)

var extractInputExtensions = []string{".mov"}

type ExtractRequest struct {
	Input   string
	Output  string // explicit output path, empty for the default name
	Bitrate int    // bits per second for CBR, 0 for the default VBR preset
}

type ExtractResult struct {
	OutputPath string
	Notes      []string
}

func ValidateExtract(req ExtractRequest) error {
	if err := media.CheckInputFile(req.Input); err != nil {
		return err
	}
	return media.CheckExtension(req.Input, extractInputExtensions...)
}

// Extract converts the single audio stream of a MOV file to MP3.
func (a *App) Extract(ctx context.Context, req ExtractRequest) (*ExtractResult, error) {
	if err := ValidateExtract(req); err != nil {
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
	}

	final := req.Output
	if final == "" {
		final, err = media.ResolveCollision(media.WithExtension(req.Input, ".mp3"))
		if err != nil {
			return nil, err
		}
	}

	out, err := reserveOutput(final)
	if err != nil {
		return nil, err
	}

	encodeReq, buildNotes := buildMP3Request(req.Input, out.Temp(), req.Bitrate, probe)
	notes = append(notes, buildNotes...)

	if err := a.encode(ctx, encodeReq); err != nil {
		out.Discard()
		return nil, err
	}
	if statOrZero(out.Temp()) == 0 {
		out.Discard()
		return nil, errors.New("ffmpeg produced no output")
	}

	if err := out.Commit(); err != nil {
		out.Discard()
		return nil, err
	}

	return &ExtractResult{OutputPath: final, Notes: notes}, nil
}
