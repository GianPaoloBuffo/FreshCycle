package schedules

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var canonicalUUIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

var supportedWeeklyDays = map[string]struct{}{
	"sunday":    {},
	"monday":    {},
	"tuesday":   {},
	"wednesday": {},
	"thursday":  {},
	"friday":    {},
	"saturday":  {},
}

type PostgresStore struct {
	db *pgxpool.Pool
}

func NewPostgresStore(db *pgxpool.Pool) PostgresStore {
	return PostgresStore{db: db}
}

func (s PostgresStore) CreateSchedule(ctx context.Context, input CreateInput) (Schedule, error) {
	name := strings.TrimSpace(input.Name)
	recurrence := strings.TrimSpace(input.Recurrence)
	garmentIDs := uniqueTrimmed(input.GarmentIDs)

	if name == "" {
		return Schedule{}, ErrNameRequired
	}

	if len(garmentIDs) == 0 {
		return Schedule{}, ErrGarmentsRequired
	}

	if !isValidRecurrence(recurrence) {
		return Schedule{}, ErrInvalidRecurrence
	}

	startsOn, err := normalizeStartDate(input.StartsOn)
	if err != nil {
		return Schedule{}, err
	}

	for _, garmentID := range garmentIDs {
		if !canonicalUUIDPattern.MatchString(garmentID) {
			return Schedule{}, ErrInvalidGarmentID
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Schedule{}, fmt.Errorf("begin create schedule: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	row := tx.QueryRow(ctx, `
		insert into public.laundry_schedules (
			user_id,
			name,
			recurrence,
			starts_on,
			reminder_enabled
		)
		values ($1, $2, $3, coalesce($4::date, current_date), $5)
		returning id, user_id, name, recurrence, to_char(starts_on, 'YYYY-MM-DD'), reminder_enabled, created_at
	`,
		strings.TrimSpace(input.UserID),
		name,
		recurrence,
		startsOn,
		input.RemindersEnabled,
	)

	var schedule Schedule
	if err := row.Scan(
		&schedule.ID,
		&schedule.UserID,
		&schedule.Name,
		&schedule.Recurrence,
		&schedule.StartsOn,
		&schedule.RemindersEnabled,
		&schedule.CreatedAt,
	); err != nil {
		return Schedule{}, fmt.Errorf("insert schedule: %w", err)
	}

	for _, garmentID := range garmentIDs {
		row := tx.QueryRow(ctx, `
			insert into public.schedule_garments (schedule_id, garment_id)
			select $1, garments.id
			from public.garments
			where garments.user_id = $2 and garments.id = $3
			returning garment_id
		`, schedule.ID, strings.TrimSpace(input.UserID), garmentID)

		var insertedGarmentID string
		if err := row.Scan(&insertedGarmentID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Schedule{}, ErrInvalidGarmentID
			}

			return Schedule{}, fmt.Errorf("insert schedule garment: %w", err)
		}

		schedule.GarmentIDs = append(schedule.GarmentIDs, insertedGarmentID)
	}

	if err := tx.Commit(ctx); err != nil {
		return Schedule{}, fmt.Errorf("commit create schedule: %w", err)
	}

	return schedule, nil
}

func (s PostgresStore) ListSchedules(ctx context.Context, userID string) ([]Schedule, error) {
	rows, err := s.db.Query(ctx, `
		select
			schedules.id,
			schedules.user_id,
			schedules.name,
			schedules.recurrence,
			to_char(schedules.starts_on, 'YYYY-MM-DD'),
			schedules.reminder_enabled,
			schedules.created_at,
			coalesce(
				array_agg(schedule_garments.garment_id::text order by schedule_garments.created_at)
					filter (where schedule_garments.garment_id is not null),
				'{}'
			)
		from public.laundry_schedules schedules
		left join public.schedule_garments schedule_garments
			on schedule_garments.schedule_id = schedules.id
		where schedules.user_id = $1
		group by schedules.id
		order by schedules.created_at desc, schedules.name asc
	`, strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	schedulesList := make([]Schedule, 0)
	for rows.Next() {
		var schedule Schedule
		if err := rows.Scan(
			&schedule.ID,
			&schedule.UserID,
			&schedule.Name,
			&schedule.Recurrence,
			&schedule.StartsOn,
			&schedule.RemindersEnabled,
			&schedule.CreatedAt,
			&schedule.GarmentIDs,
		); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}

		schedulesList = append(schedulesList, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schedules: %w", err)
	}

	return schedulesList, nil
}

func (s PostgresStore) DeleteSchedule(ctx context.Context, userID string, scheduleID string) error {
	trimmedID := strings.TrimSpace(scheduleID)
	if !canonicalUUIDPattern.MatchString(trimmedID) {
		return ErrInvalidID
	}

	tag, err := s.db.Exec(ctx, `
		delete from public.laundry_schedules
		where user_id = $1 and id = $2
	`, strings.TrimSpace(userID), trimmedID)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrScheduleNotFound
	}

	return nil
}

func uniqueTrimmed(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		if _, ok := seen[trimmed]; ok {
			continue
		}

		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return normalized
}

func isValidRecurrence(recurrence string) bool {
	switch recurrence {
	case "daily", "fortnightly":
		return true
	}

	if !strings.HasPrefix(recurrence, "weekly:") {
		return false
	}

	weekday := strings.TrimSpace(strings.TrimPrefix(recurrence, "weekly:"))
	_, ok := supportedWeeklyDays[weekday]
	return ok
}

func normalizeStartDate(value *string) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*value)
	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return nil, ErrInvalidStartDate
	}

	return &trimmed, nil
}
