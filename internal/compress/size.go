package compress

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Decimal units: 1 kb = 1000 bytes
const (
	Kilobyte = 1_000
	Megabyte = 1_000_000
	Gigabyte = 1_000_000_000
)

var sizePattern = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s?(kb|mb|gb)$`)

// ParseSize parses decimal sizes like "80mb" or "1.5gb", case-insensitive,
// and rounds to whole bytes
func ParseSize(s string) (int64, error) {
	m := sizePattern.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, fmt.Errorf("%w: %q (expected forms: 80mb, 500kb, 1.5gb)", ErrInvalidSize, s)
	}

	value, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidSize, s)
	}

	var unit int64
	switch strings.ToLower(m[2]) {
	case "kb":
		unit = Kilobyte
	case "mb":
		unit = Megabyte
	default:
		unit = Gigabyte
	}

	// Compare in float first: huge inputs overflow int64, and the conversion
	// saturates differently per architecture
	bytes := value * float64(unit)
	if bytes >= math.MaxInt64 {
		return 0, fmt.Errorf("%w: %q is too large", ErrInvalidSize, s)
	}

	rounded := int64(math.Round(bytes))
	if rounded <= 0 {
		return 0, fmt.Errorf("%w: %q must be greater than zero", ErrInvalidSize, s)
	}
	return rounded, nil
}
