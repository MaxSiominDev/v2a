package ffmpeg

import "errors"

var (
	ErrFFmpegNotFound  = errors.New("ffmpeg is required but was not found in PATH.\nInstall it with: brew install ffmpeg")
	ErrFFprobeNotFound = errors.New("ffprobe is required but was not found in PATH.\nInstall it with: brew install ffmpeg")
)
