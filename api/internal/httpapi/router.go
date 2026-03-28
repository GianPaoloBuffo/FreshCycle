package httpapi

import (
	"net/http"
	"time"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter() http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(30 * time.Second))

	router.Get("/health", handlers.Health)

	return router
}
