package schedules

import (
	"context"
	"time"
)

type CreateInput struct {
	UserID           string
	Name             string
	Recurrence       string
	GarmentIDs       []string
	RemindersEnabled bool
}

type Schedule struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Name             string    `json:"name"`
	Recurrence       string    `json:"recurrence"`
	GarmentIDs       []string  `json:"garment_ids"`
	RemindersEnabled bool      `json:"reminders_enabled"`
	CreatedAt        time.Time `json:"created_at"`
}

type Store interface {
	CreateSchedule(ctx context.Context, input CreateInput) (Schedule, error)
	DeleteSchedule(ctx context.Context, userID string, scheduleID string) error
	ListSchedules(ctx context.Context, userID string) ([]Schedule, error)
}
