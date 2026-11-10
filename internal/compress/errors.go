package compress

import "errors"

var (
	ErrInvalidBitrate  = errors.New("invalid bitrate")
	ErrInvalidSize     = errors.New("invalid size")
	ErrTargetTooSmall  = errors.New("target size is too small for this audio")
	ErrBitrateNotLower = errors.New("requested bitrate is not lower than the source bitrate")
	ErrBitrateTooLow   = errors.New("requested bitrate is below the lowest supported (32k)")
)
