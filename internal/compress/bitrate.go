package compress

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	// MinBitrate is the floor for any encode: below it the audio is unusable,
	// so an unreachable target fails instead.
	MinBitrate = 32_000
	// MaxBitrate matches the libmp3lame limit.
	MaxBitrate  = 320_000
	MaxAttempts = 8
)

var bitratePattern = regexp.MustCompile(`(?i)^(\d+)(\s?(?:k|kb|kbps))?$`)

// ParseBitrate parses "128k", "192KBPS" or a bare bits-per-second "96000".
func ParseBitrate(s string) (int, error) {
	m := bitratePattern.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, fmt.Errorf("%w: %q (expected forms: 128k, 192k, 96000)", ErrInvalidBitrate, s)
	}

	value, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidBitrate, s)
	}

	switch strings.ToLower(strings.TrimSpace(m[2])) {
	case "k", "kb", "kbps":
		value *= 1000
	}

	if value <= 0 || value > MaxBitrate {
		return 0, fmt.Errorf("%w: %d must be between 1 and %d bits per second", ErrInvalidBitrate, value, MaxBitrate)
	}
	return value, nil
}

// BitrateLadder lists the constant bitrates libmp3lame accepts for the sample
// rate, descending. MP3 frames come in fixed sizes, so CBR output only ever
// lands on these steps.
func BitrateLadder(sampleRate int) []int {
	mpeg1 := []int{320, 256, 224, 192, 160, 144, 128, 112, 96, 80, 64, 56, 48, 40, 32}
	mpeg2 := []int{160, 144, 128, 112, 96, 80, 64, 56, 48, 40, 32, 24, 16, 8}

	if sampleRate > 0 && sampleRate < 32000 {
		return kilobits(mpeg2)
	}
	return kilobits(mpeg1)
}

func kilobits(steps []int) []int {
	out := make([]int, len(steps))
	for i, s := range steps {
		out[i] = s * 1000
	}
	return out
}

// FloorBitrate maps bps down to the nearest supported step, 0 if below all.
func FloorBitrate(bps, sampleRate int) int {
	for _, step := range BitrateLadder(sampleRate) {
		if step <= bps {
			return step
		}
	}
	return 0
}

// FitBitrate returns the highest supported bitrate whose payload fits in
// targetBytes alongside overheadBytes, for audio of the given duration.
func FitBitrate(targetBytes, overheadBytes int64, durationSeconds float64, sampleRate int) (int, error) {
	if durationSeconds <= 0 {
		return 0, fmt.Errorf("%w: unknown audio duration", ErrTargetTooSmall)
	}

	bps := int(float64(targetBytes-overheadBytes) * 8 / durationSeconds)
	bps = FloorBitrate(bps, sampleRate)
	if bps < MinBitrate {
		return 0, ErrTargetTooSmall
	}
	return bps, nil
}
