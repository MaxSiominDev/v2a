package compress

import (
	"errors"
	"testing"
)

func TestParseBitrate(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"128k", 128_000, false},
		{"128K", 128_000, false},
		{"128kb", 128_000, false},
		{"128kbps", 128_000, false},
		{"128 KBPS", 128_000, false},
		{"96000", 96_000, false},
		{"175k", 175_000, false},

		{"", 0, true},
		{"abc", 0, true},
		{"1.5k", 0, true},
		{"128 mb", 0, true},
		{"400k", 0, true},
		{"0", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseBitrate(tt.input)
		if tt.wantErr {
			if !errors.Is(err, ErrInvalidBitrate) {
				t.Errorf("ParseBitrate(%q) error = %v, want ErrInvalidBitrate", tt.input, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseBitrate(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseBitrate(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestBitrateLadder(t *testing.T) {
	mpeg1 := BitrateLadder(44100)
	if mpeg1[0] != 320_000 || mpeg1[len(mpeg1)-1] != 32_000 {
		t.Errorf("MPEG-1 ladder ends = %d..%d, want 320000..32000", mpeg1[0], mpeg1[len(mpeg1)-1])
	}

	mpeg2 := BitrateLadder(22050)
	if mpeg2[len(mpeg2)-1] != 8_000 {
		t.Errorf("MPEG-2 ladder minimum = %d, want 8000", mpeg2[len(mpeg2)-1])
	}

	// Unknown sample rate falls back to the conservative MPEG-1 ladder.
	if got := BitrateLadder(0); got[len(got)-1] != 32_000 {
		t.Errorf("default ladder minimum = %d, want 32000", got[len(got)-1])
	}
}

func TestFloorBitrate(t *testing.T) {
	tests := []struct {
		bps        int
		sampleRate int
		want       int
	}{
		{79022, 44100, 64_000},
		{80_000, 44100, 80_000},
		{1_000_000, 44100, 320_000},
		{1_000, 44100, 0},
		{90_000, 22050, 80_000},
	}

	for _, tt := range tests {
		if got := FloorBitrate(tt.bps, tt.sampleRate); got != tt.want {
			t.Errorf("FloorBitrate(%d, %d) = %d, want %d", tt.bps, tt.sampleRate, got, tt.want)
		}
	}
}

func TestFitBitrate(t *testing.T) {
	// 10 seconds of audio in a 100000 byte budget with 1222 bytes overhead:
	// (100000-1222)*8/10 = 79022 bps, floored to the nearest MP3 step
	got, err := FitBitrate(100_000, 1_222, 10, 44_100)
	if err != nil {
		t.Fatalf("FitBitrate unexpected error: %v", err)
	}
	if got != 64_000 {
		t.Errorf("FitBitrate = %d, want 64000", got)
	}

	// Overshooting the encoder maximum clamps to it
	got, err = FitBitrate(1_000_000_000, 0, 1, 44_100)
	if err != nil {
		t.Fatalf("FitBitrate unexpected error: %v", err)
	}
	if got != MaxBitrate {
		t.Errorf("FitBitrate = %d, want %d", got, MaxBitrate)
	}

	// A budget that even MinBitrate cannot satisfy fails
	if _, err := FitBitrate(10_000, 0, 10, 44_100); !errors.Is(err, ErrTargetTooSmall) {
		t.Errorf("FitBitrate error = %v, want ErrTargetTooSmall", err)
	}

	// Negative overhead (target smaller than the tags alone) fails too
	if _, err := FitBitrate(1_000, 2_000, 10, 44_100); !errors.Is(err, ErrTargetTooSmall) {
		t.Errorf("FitBitrate error = %v, want ErrTargetTooSmall", err)
	}

	if _, err := FitBitrate(100_000, 0, 0, 44_100); !errors.Is(err, ErrTargetTooSmall) {
		t.Errorf("FitBitrate with zero duration error = %v, want ErrTargetTooSmall", err)
	}
}
