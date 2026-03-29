package main

import (
	"log"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/app"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load(".env.local")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("app bootstrap error: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("api server stopped: %v", err)
	}
}
