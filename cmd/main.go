package main

import (
	"log"

	"github.com/dione-docs-backend/internal/app"
)

func main() {
	application, err := app.NewApplication()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
