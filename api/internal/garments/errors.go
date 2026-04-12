package garments

import "errors"

var (
	ErrNameRequired              = errors.New("garment name is required")
	ErrWashTemperatureOutOfRange = errors.New("wash temperature must be between 0 and 95")
	ErrInvalidID                 = errors.New("garment id must be a valid uuid")
	ErrInvalidLabelImagePath     = errors.New("label image path must stay within the authenticated user's label folder")
)
