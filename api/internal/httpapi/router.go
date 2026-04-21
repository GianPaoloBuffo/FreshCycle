package httpapi

import (
	"net/http"
	"time"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/auth"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/garments"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi/handlers"
	httpmiddleware "github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi/middleware"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/schedules"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(parser labelparser.Parser, garmentStore garments.Store, allowedOrigins []string, validator auth.Validator, scheduleStores ...schedules.Store) http.Handler {
	router := chi.NewRouter()
	var scheduleStore schedules.Store
	if len(scheduleStores) > 0 {
		scheduleStore = scheduleStores[0]
	}

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(30 * time.Second))
	router.Use(httpmiddleware.CORS(httpmiddleware.CORSConfig{
		AllowedOrigins: allowedOrigins,
	}))

	router.Get("/health", handlers.Health)
	router.With(httpmiddleware.RequireAuth(validator)).Post("/garments/parse-label", handlers.ParseLabel(parser))
	router.With(httpmiddleware.RequireAuth(validator)).Get("/garments", handlers.ListGarments(garmentStore))
	router.With(httpmiddleware.RequireAuth(validator)).Get("/garments/{garmentID}", handlers.GetGarment(garmentStore))
	router.With(httpmiddleware.RequireAuth(validator)).Post("/garments", handlers.CreateGarment(garmentStore))
	router.With(httpmiddleware.RequireAuth(validator)).Get("/schedules", handlers.ListSchedules(scheduleStore))
	router.With(httpmiddleware.RequireAuth(validator)).Post("/schedules", handlers.CreateSchedule(scheduleStore))
	router.With(httpmiddleware.RequireAuth(validator)).Delete("/schedules/{scheduleID}", handlers.DeleteSchedule(scheduleStore))

	return router
}
