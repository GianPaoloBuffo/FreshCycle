package garments

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

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

	if input.WashTemperatureC != nil && (*input.WashTemperatureC < 0 || *input.WashTemperatureC > 95) {
		return Garment{}, ErrWashTemperatureOutOfRange
	}

	row := s.db.QueryRow(ctx, `
		insert into public.garments (
			user_id,
			name,
			category,
			primary_color,
			wash_temperature_c,
			care_instructions,
			label_image_path
		)
		values ($1, $2, $3, $4, $5, $6, $7)
		returning id, user_id, name, category, primary_color, wash_temperature_c, care_instructions, label_image_path
	`,
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
