package garments

import "errors"

var (
	ErrNameRequired              = errors.New("garment name is required")
	ErrWashTemperatureOutOfRange = errors.New("wash temperature must be between 0 and 95")
)
