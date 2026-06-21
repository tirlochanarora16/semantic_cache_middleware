package main

import (
	"context"
	"log"

	"example.com/database"
	"example.com/llm"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalf("Error loading env variables variables %v", err)
		return
	}

	err = database.InitializeRedisDB()

	if err != nil {
		log.Fatalf("Error connecting to Redis %v", err)
		return
	}

	pool, err := database.InitializePostgresDB()

	if err != nil {
		log.Fatalf("Error connecting to PSQL %v", err)
		return
	}

	defer pool.Close()

	client, err := llm.NewClient()

	if err != nil {
		log.Fatalf("Unable to initalize OpenAI client %v", err)
		return
	}

	aiService := llm.NewClientService(client)
	aiService.RunAllServices(context.Background())

}
