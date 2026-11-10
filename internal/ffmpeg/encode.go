package ffmpeg

import (
	"fmt"
	"strconv"
)

// MP3Request is one MP3 encode. Stream, quality and metadata decisions are
// made upstream; Args only translates them into ffmpeg arguments, passed
// directly to the process, never through a shell
type MP3Request struct {
	Input  string
	Output string

	// Bitrate in bits per second for CBR; zero means the VBR preset
	Bitrate int

	// Downmix folds more than two channels to stereo
	Downmix bool
	// Resample caps the sample rate at 48 kHz
	Resample bool

	// CoverIndex is the input stream index of the attached picture, -1 if none
	CoverIndex int

	MapGlobalMetadata bool
	// ClearMetadataKeys empties tags that must not leak into the output
	ClearMetadataKeys []string
	// FallbackTitle sets a global title when the source has none
	FallbackTitle string
}

func (r MP3Request) Args() []string {
	args := []string{
		"-nostdin",
		"-v", "error",
		"-y",
		"-i", r.Input,
		"-map", "0:a:0",
	}

	if r.CoverIndex >= 0 {
		args = append(args,
			"-map", fmt.Sprintf("0:%d", r.CoverIndex),
			"-c:v", "copy",
			"-disposition:v", "attached_pic",
		)
	}

	args = append(args, "-c:a", "libmp3lame")
	if r.Bitrate > 0 {
		args = append(args, "-b:a", strconv.Itoa(r.Bitrate))
	} else {
		args = append(args, "-q:a", "0")
	}

	if r.Downmix {
		args = append(args, "-ac", "2")
	}
	if r.Resample {
		args = append(args, "-ar", "48000")
	}

	if r.MapGlobalMetadata {
		args = append(args, "-map_metadata", "0")
	}
	for _, key := range r.ClearMetadataKeys {
		args = append(args, "-metadata", key+"=")
	}
	if r.FallbackTitle != "" {
		args = append(args, "-metadata", "title="+r.FallbackTitle)
	}

	return append(args, r.Output)
}
