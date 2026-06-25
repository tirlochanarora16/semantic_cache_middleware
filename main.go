package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"example.com/api"
	"example.com/database"
	"example.com/llm"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		// to work seemlessly in docker container
		log.Println("No .env file found, using environment variables")
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
	handler := api.NewHandler(aiService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/generate", handler.Generate)

	server := &http.Server{
		Addr:              ":3000",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("HTTP server listening on port 3000")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server failed: %v", err)
	}

}
