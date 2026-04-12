package garments

import "context"

type CreateInput struct {
	UserID           string
	Name             string
	Category         *string
	PrimaryColor     *string
	WashTemperatureC *int
	CareInstructions []string
	LabelImagePath   *string
}

type Garment struct {
	ID               string   `json:"id"`
	UserID           string   `json:"user_id"`
	Name             string   `json:"name"`
	Category         *string  `json:"category"`
	PrimaryColor     *string  `json:"primary_color"`
	WashTemperatureC *int     `json:"wash_temperature_c"`
	CareInstructions []string `json:"care_instructions"`
	LabelImagePath   *string  `json:"label_image_path"`
}

type Store interface {
	CreateGarment(ctx context.Context, input CreateInput) (Garment, error)
}
