package compress

import (
	"errors"
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr error
	}{
		{"80mb", 80_000_000, nil},
		{"80MB", 80_000_000, nil},
		{"80Mb", 80_000_000, nil},
		{"500kb", 500_000, nil},
		{"500KB", 500_000, nil},
		{"1gb", 1_000_000_000, nil},
		{"1.5gb", 1_500_000_000, nil},
		{"0.5mb", 500_000, nil},
		{"80 mb", 80_000_000, nil},
		{"  80mb  ", 80_000_000, nil},

		{"80", 0, ErrInvalidSize},
		{"80kbx", 0, ErrInvalidSize},
		{"mb", 0, ErrInvalidSize},
		{"", 0, ErrInvalidSize},
		{"-80mb", 0, ErrInvalidSize},
		{"0mb", 0, ErrInvalidSize},
		{"80 mb extra", 0, ErrInvalidSize},
	}

	for _, tt := range tests {
		got, err := ParseSize(tt.input)
		if tt.wantErr != nil {
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ParseSize(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSize(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseSize(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
