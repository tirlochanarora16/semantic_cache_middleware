package main

import (
	"context"
	"log"
	"time"

	"example.com/database"
	"example.com/llm"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading env variables variables %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := database.InitializeRedisDB(); err != nil {
		log.Fatalf("Redis checkup failed %v", err)
		return
	}

	if err := database.CheckRedisStack(ctx); err != nil {
		log.Fatalf("Redis stack checkup failed %v", err)
		return
	}

	if err := database.EnsureVectorIndex(ctx); err != nil {
		log.Fatalf("Redis index startup check failed %v", err)
		return
	}

	pool, err := database.InitializePostgresDB()

	if err != nil {
		log.Fatalf("Postgres checkup failed %v", err)
		return
	}

	defer pool.Close()

	client, err := llm.NewClient()

	if err != nil {
		log.Fatalf("Unable to initalize OpenAI client %v", err)
		return
	}

	if err := llm.CheckOpenAI(ctx, client); err != nil {
		log.Fatalf("OpenAI startup check failed: %v", err)
	}

	log.Println("All Startup checks passed")

	aiService := llm.NewClientService(client)
	aiService.RunAllServices(context.Background())

}
