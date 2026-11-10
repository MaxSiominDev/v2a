package application

import (
	"context"

	"github.com/MaxSiominDev/v2a/internal/media"
)

// Info accepts any file ffprobe understands, unlike extract
func (a *App) Info(ctx context.Context, path string) (*media.ProbeResult, error) {
	if err := media.CheckInputFile(path); err != nil {
		return nil, err
	}
	return a.probe(ctx, path)
}
