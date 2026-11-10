package ffmpeg

import (
	"slices"
	"testing"
)

func TestMP3RequestArgsDefaultVBR(t *testing.T) {
	req := MP3Request{Input: "in.mov", Output: "out.mp3", CoverIndex: -1, MapGlobalMetadata: true}

	want := []string{
		"-nostdin", "-v", "error", "-y",
		"-i", "in.mov",
		"-map", "0:a:0",
		"-c:a", "libmp3lame",
		"-q:a", "0",
		"-map_metadata", "0",
		"out.mp3",
	}
	if got := req.Args(); !slices.Equal(got, want) {
		t.Errorf("Args() = %v, want %v", got, want)
	}
}

func TestMP3RequestArgsCBRWithCover(t *testing.T) {
	req := MP3Request{
		Input:             "in.mp3",
		Output:            "out.mp3",
		Bitrate:           128_000,
		CoverIndex:        2,
		MapGlobalMetadata: true,
	}

	want := []string{
		"-nostdin", "-v", "error", "-y",
		"-i", "in.mp3",
		"-map", "0:a:0",
		"-map", "0:2",
		"-c:v", "copy",
		"-disposition:v", "attached_pic",
		"-c:a", "libmp3lame",
		"-b:a", "128000",
		"-map_metadata", "0",
		"out.mp3",
	}
	if got := req.Args(); !slices.Equal(got, want) {
		t.Errorf("Args() = %v, want %v", got, want)
	}
}

func TestMP3RequestArgsAdjustmentsAndMetadata(t *testing.T) {
	req := MP3Request{
		Input:             "in.mov",
		Output:            "out.mp3",
		Downmix:           true,
		Resample:          true,
		CoverIndex:        -1,
		MapGlobalMetadata: true,
		ClearMetadataKeys: []string{"major_brand", "minor_version"},
		FallbackTitle:     "Stream Title",
	}

	got := req.Args()
	want := []string{
		"-nostdin", "-v", "error", "-y",
		"-i", "in.mov",
		"-map", "0:a:0",
		"-c:a", "libmp3lame",
		"-q:a", "0",
		"-ac", "2",
		"-ar", "48000",
		"-map_metadata", "0",
		"-metadata", "major_brand=",
		"-metadata", "minor_version=",
		"-metadata", "title=Stream Title",
		"out.mp3",
	}
	if !slices.Equal(got, want) {
		t.Errorf("Args() = %v, want %v", got, want)
	}
}
