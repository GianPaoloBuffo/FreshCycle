package main

import (
	"log"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/app"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/config"
)

func main() {
	cfg := config.Load()

	application := app.New(cfg)

	if err := application.Run(); err != nil {
		log.Fatalf("api server stopped: %v", err)
	}
}
