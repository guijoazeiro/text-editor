package main

import (
	"log"

	"github.com/guijoazeiro/text-editor/tree/main/server/internal/app"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
