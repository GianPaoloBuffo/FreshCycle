package garments

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var canonicalUUIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type PostgresStore struct {
	db *pgxpool.Pool
}

func NewPostgresStore(db *pgxpool.Pool) PostgresStore {
	return PostgresStore{db: db}
}

func (s PostgresStore) CreateGarment(ctx context.Context, input CreateInput) (Garment, error) {
	if strings.TrimSpace(input.Name) == "" {
		return Garment{}, ErrNameRequired
	}

	if input.ID != nil && !canonicalUUIDPattern.MatchString(strings.TrimSpace(*input.ID)) {
		return Garment{}, ErrInvalidID
	}

	if input.WashTemperatureC != nil && (*input.WashTemperatureC < 0 || *input.WashTemperatureC > 95) {
		return Garment{}, ErrWashTemperatureOutOfRange
	}

	if input.LabelImagePath != nil {
		expectedPrefix := strings.TrimSpace(input.UserID) + "/labels/"
		if !strings.HasPrefix(strings.TrimSpace(*input.LabelImagePath), expectedPrefix) {
			return Garment{}, ErrInvalidLabelImagePath
		}
	}

	row := s.db.QueryRow(ctx, `
		insert into public.garments (
			id,
			user_id,
			name,
			category,
			primary_color,
			wash_temperature_c,
			care_instructions,
			label_image_path
		)
		values (coalesce($1::uuid, gen_random_uuid()), $2, $3, $4, $5, $6, $7, $8)
		returning id, user_id, name, category, primary_color, wash_temperature_c, care_instructions, label_image_path
	`,
		input.ID,
		input.UserID,
		strings.TrimSpace(input.Name),
		input.Category,
		input.PrimaryColor,
		input.WashTemperatureC,
		input.CareInstructions,
		input.LabelImagePath,
	)

	var garment Garment
	if err := row.Scan(
		&garment.ID,
		&garment.UserID,
		&garment.Name,
		&garment.Category,
		&garment.PrimaryColor,
		&garment.WashTemperatureC,
		&garment.CareInstructions,
		&garment.LabelImagePath,
	); err != nil {
		return Garment{}, fmt.Errorf("insert garment: %w", err)
	}

	return garment, nil
}

func (s PostgresStore) GetGarment(ctx context.Context, userID string, garmentID string) (Garment, error) {
	trimmedID := strings.TrimSpace(garmentID)
	if !canonicalUUIDPattern.MatchString(trimmedID) {
		return Garment{}, ErrInvalidID
	}

	row := s.db.QueryRow(ctx, `
		select id, user_id, name, category, primary_color, wash_temperature_c, care_instructions, label_image_path
		from public.garments
		where user_id = $1 and id = $2
	`, strings.TrimSpace(userID), trimmedID)

	var garment Garment
	if err := row.Scan(
		&garment.ID,
		&garment.UserID,
		&garment.Name,
		&garment.Category,
		&garment.PrimaryColor,
		&garment.WashTemperatureC,
		&garment.CareInstructions,
		&garment.LabelImagePath,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Garment{}, ErrGarmentNotFound
		}

		return Garment{}, fmt.Errorf("get garment: %w", err)
	}

	return garment, nil
}

func (s PostgresStore) ListGarments(ctx context.Context, userID string, options ListOptions) ([]Garment, error) {
	query := `
		select id, user_id, name, category, primary_color, wash_temperature_c, care_instructions, label_image_path
		from public.garments
		where user_id = $1
	`

	args := []any{strings.TrimSpace(userID)}
	if options.Category != nil {
		query += ` and lower(category) = lower($2)`
		args = append(args, strings.TrimSpace(*options.Category))
	}

	query += " order by "
	switch options.SortBy {
	case "name":
		query += "name"
	default:
		query += "created_at"
	}

	switch options.Order {
	case "asc":
		query += " asc"
	default:
		query += " desc"
	}

	query += ", name asc"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list garments: %w", err)
	}
	defer rows.Close()

	garmentsList := make([]Garment, 0)
	for rows.Next() {
		var garment Garment
		if err := rows.Scan(
			&garment.ID,
			&garment.UserID,
			&garment.Name,
			&garment.Category,
			&garment.PrimaryColor,
			&garment.WashTemperatureC,
			&garment.CareInstructions,
			&garment.LabelImagePath,
		); err != nil {
			return nil, fmt.Errorf("scan garment: %w", err)
		}

		garmentsList = append(garmentsList, garment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate garments: %w", err)
	}

	return garmentsList, nil
}
