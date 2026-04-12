package httpapi

import (
	"net/http"
	"time"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi/handlers"
	httpmiddleware "github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi/middleware"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(parser labelparser.Parser, allowedOrigins []string) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(30 * time.Second))
	router.Use(httpmiddleware.CORS(httpmiddleware.CORSConfig{
		AllowedOrigins: allowedOrigins,
	}))

	router.Get("/health", handlers.Health)
	router.Post("/garments/parse-label", handlers.ParseLabel(parser))

	return router
}
