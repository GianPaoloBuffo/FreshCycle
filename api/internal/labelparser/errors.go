package labelparser

import "errors"

var (
	ErrImageRequired         = errors.New("label image is required")
	ErrUnsupportedImageType  = errors.New("unsupported label image type")
	ErrProviderUnavailable   = errors.New("label parser provider is unavailable")
	ErrUpstreamParseRejected = errors.New("label parser provider could not parse the image")
)
