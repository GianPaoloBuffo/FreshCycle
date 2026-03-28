package handlers

import (
	"encoding/json"
	"net/http"
)

type healthResponse struct {
	Status string `json:"status"`
}

func Health(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)

	response := healthResponse{Status: "ok"}

	_ = json.NewEncoder(writer).Encode(response)
}
