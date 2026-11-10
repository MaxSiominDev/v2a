package ffmpeg

import "os/exec"

type ToolLocator struct {
	FFmpegPath  string
	FFprobePath string
}

func NewToolLocator() *ToolLocator {
	return &ToolLocator{FFmpegPath: "ffmpeg", FFprobePath: "ffprobe"}
}

// Locate returns the resolved paths of both tools
func (l *ToolLocator) Locate() (ffmpegPath, ffprobePath string, err error) {
	ffmpegPath, err = exec.LookPath(l.FFmpegPath)
	if err != nil {
		return "", "", ErrFFmpegNotFound
	}

	ffprobePath, err = exec.LookPath(l.FFprobePath)
	if err != nil {
		return "", "", ErrFFprobeNotFound
	}
	return ffmpegPath, ffprobePath, nil
}
