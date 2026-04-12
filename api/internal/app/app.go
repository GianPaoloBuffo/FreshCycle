package app

import (
	"log"
	"net/http"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/config"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	config config.Config
	db     *pgxpool.Pool
	server *http.Server
}

func New(cfg config.Config) (*App, error) {
	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	parser, err := labelparser.NewParser(cfg)
	if err != nil {
		db.Close()
		return nil, err
	}

	router := httpapi.NewRouter(parser, cfg.AllowedOrigins)

	return &App{
		config: cfg,
		db:     db,
		server: &http.Server{
			Addr:    cfg.Address(),
			Handler: router,
		},
	}, nil
}

func (a *App) Run() error {
	defer a.db.Close()

	log.Printf("freshcycle api listening on %s", a.config.Address())
	log.Printf("supabase postgres connection %s", a.config.RedactedDatabaseURL())

	return a.server.ListenAndServe()
}
