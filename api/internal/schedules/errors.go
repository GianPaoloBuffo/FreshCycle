package schedules

import "errors"

var (
	ErrNameRequired      = errors.New("schedule name is required")
	ErrGarmentsRequired  = errors.New("at least one garment is required")
	ErrInvalidID         = errors.New("schedule id must be a valid uuid")
	ErrInvalidGarmentID  = errors.New("garment id must be a valid owned garment uuid")
	ErrInvalidRecurrence = errors.New("recurrence must be daily, fortnightly, or weekly:<weekday>")
	ErrInvalidStartDate  = errors.New("start date must use YYYY-MM-DD format")
	ErrScheduleNotFound  = errors.New("schedule not found")
)
