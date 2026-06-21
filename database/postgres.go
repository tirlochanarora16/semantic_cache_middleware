package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"example.com/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openai/openai-go/v3"
)

var psqlClinet *db.Queries

func InitializePostgresDB() (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(os.Getenv("PSQL_CONN_STR"))

	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 1
	config.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)

	if err != nil {
		return nil, err
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	psqlClinet = db.New(pool)

	return pool, nil
}

func SaveResponseToPostgres(ctx context.Context, chatCompletion *openai.ChatCompletion, userInput string, latency int32) error {
	_, err := psqlClinet.CreateLog(ctx, db.CreateLogParams{
		Prompt:    userInput,
		Response:  chatCompletion.Choices[0].Message.Content,
		LatencyMs: latency,
	})

	return err
}
