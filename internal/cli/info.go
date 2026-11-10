package cli

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MaxSiominDev/v2a/internal/application"
	"github.com/MaxSiominDev/v2a/internal/media"
)

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <input>",
		Short: "Show information about a media file",
		Args:  cobra.ExactArgs(1),
		Example: `  v2a info movie.mov
  v2a info song.mp3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := application.New()
			if err != nil {
				return err
			}

			probe, err := app.Info(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			printInfo(args[0], probe)
			return nil
		},
	}
}

func printInfo(path string, probe *media.ProbeResult) {
	printFileInfo(path, probe)
	if video := videoStreams(probe); len(video) > 0 {
		fmt.Println()
		printVideoInfo(video)
	}
	if audio := probe.AudioStreams(); len(audio) > 0 {
		fmt.Println()
		printAudioInfo(audio)
	}
	if metadata := probe.Metadata(); len(metadata) > 0 || probe.AttachedPicture() != nil {
		fmt.Println()
		printMetadataInfo(probe)
	}
}

func printFileInfo(path string, probe *media.ProbeResult) {
	fmt.Println("File")
	fmt.Printf("  Path:      %s\n", path)
	if size, err := probe.SizeBytes(); err == nil {
		fmt.Printf("  Size:      %s\n", humanSize(size))
	}
	if seconds, err := probe.DurationSeconds(); err == nil && seconds > 0 {
		fmt.Printf("  Duration:  %s\n", media.DurationString(seconds))
	}
}

func printVideoInfo(video []media.Stream) {
	fmt.Println("Video")
	for _, s := range video {
		fmt.Printf("  Codec:       %s\n", s.CodecName)
		if s.Width > 0 && s.Height > 0 {
			fmt.Printf("  Resolution:  %dx%d\n", s.Width, s.Height)
		}
		if fps := frameRate(s.AvgFrameRate); fps != "" {
			fmt.Printf("  FPS:         %s\n", fps)
		}
	}
}

func printAudioInfo(audio []media.Stream) {
	fmt.Println("Audio")
	for _, s := range audio {
		fmt.Printf("  Codec:       %s\n", s.CodecName)
		if s.BitRate != "" {
			if bps, err := strconv.Atoi(s.BitRate); err == nil {
				fmt.Printf("  Bitrate:     %d kb/s\n", bps/1000)
			}
		}
		if s.SampleRate != "" {
			fmt.Printf("  Sample rate: %s Hz\n", s.SampleRate)
		}
		if s.Channels > 0 {
			fmt.Printf("  Channels:    %d\n", s.Channels)
		}
		if lang, ok := s.Language(); ok && lang != "" {
			fmt.Printf("  Language:    %s\n", lang)
		}
		if title, ok := s.Title(); ok && title != "" {
			fmt.Printf("  Title:       %s\n", title)
		}
	}
}

func printMetadataInfo(probe *media.ProbeResult) {
	fmt.Println("Metadata")
	metadata := probe.Metadata()
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("  %s: %s\n", key, metadata[key])
	}

	cover := "no"
	if probe.AttachedPicture() != nil {
		cover = "yes"
	}
	fmt.Printf("  Cover art: %s\n", cover)
}

func videoStreams(probe *media.ProbeResult) []media.Stream {
	var out []media.Stream
	for _, s := range probe.Streams {
		if s.CodecType == "video" && s.Disposition.AttachedPic != 1 {
			out = append(out, s)
		}
	}
	return out
}

func frameRate(ratio string) string {
	num, den, ok := strings.Cut(ratio, "/")
	if !ok {
		return ratio
	}

	n, errN := strconv.ParseFloat(num, 64)
	d, errD := strconv.ParseFloat(den, 64)
	if errN != nil || errD != nil || d == 0 {
		return ""
	}

	return strconv.FormatFloat(n/d, 'f', 2, 64)
}

func humanSize(bytes int64) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	value := float64(bytes)
	units := []string{"kB", "MB", "GB", "TB"}
	for _, u := range units {
		value /= unit
		if value < unit {
			return strings.TrimSuffix(strconv.FormatFloat(value, 'f', 1, 64), ".0") + " " + u
		}
	}
	return fmt.Sprintf("%.0f PB", value/math.Pow(unit, float64(len(units))))
}
