package app

import (
	"log"
	"net/http"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/config"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi"
)

type App struct {
	config config.Config
	server *http.Server
}

func New(cfg config.Config) *App {
	router := httpapi.NewRouter()

	return &App{
		config: cfg,
		server: &http.Server{
			Addr:    cfg.Address(),
			Handler: router,
		},
	}
}

func (a *App) Run() error {
	log.Printf("freshcycle api listening on %s", a.config.Address())

	return a.server.ListenAndServe()
}
