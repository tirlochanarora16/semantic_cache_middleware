package database

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openai/openai-go/v3"
	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

type VectorSearchResult struct {
	Key      string
	Response string
	Score    float64
	Found    bool
}

func InitializeRedisDB() error {
	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Username: "",
		Password: "",
		DB:       0,
		Protocol: 2,
	})

	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		return err
	}

	redisClient = client

	return nil
}

func EnsureVectorIndex(ctx context.Context) error {
	_, err := redisClient.Do(ctx, "FT.INFO", "idx:cache").Result()

	if err == nil {
		return nil
	}

	if !strings.Contains(err.Error(), "No such index") && !strings.Contains(err.Error(), "Unknown index name") {
		return err
	}

	_, err = redisClient.Do(
		ctx,
		"FT.CREATE",
		"idx:cache",
		"ON", "HASH",
		"PREFIX", "1", "cache:",
		"SCHEMA",
		"prompt", "TEXT",
		"response", "TEXT",
		"created_at", "NUMERIC",
		"embedding", "VECTOR", "HNSW", "6",
		"TYPE", "FLOAT32",
		"DIM", "1536",
		"DISTANCE_METRIC", "COSINE",
	).Result()

	if err != nil {
		return err
	}

	return err
}

func DoVectorSearch(ctx context.Context, vectorBytes []byte) (*VectorSearchResult, error) {
	res, err := redisClient.Do(ctx, "FT.SEARCH", "idx:cache", "*=>[KNN 1 @embedding $vec AS score]", "PARAMS", 2, "vec", vectorBytes, "SORTBY", "score", "DIALECT", 2, "LIMIT", 0, 1).Result()

	if err != nil {
		return &VectorSearchResult{}, err
	}

	parts, ok := res.([]interface{})

	if !ok || len(parts) < 1 {
		return &VectorSearchResult{Found: false}, nil
	}

	total, ok := parts[0].(int64)

	if !ok || total == 0 || len(parts) < 3 {
		return &VectorSearchResult{Found: false}, nil
	}

	out := &VectorSearchResult{Found: true}

	if key, ok := parts[1].(string); ok {
		out.Key = key
	}

	attrs, ok := parts[2].([]interface{})

	if !ok {
		return out, nil
	}

	for i := 0; i+1 < len(attrs); i += 2 {
		field, _ := attrs[i].(string)
		value := fmt.Sprint(attrs[i+1])

		switch field {
		case "response":
			out.Response = value
		case "score":
			if score, err := strconv.ParseFloat(value, 64); err == nil {
				out.Score = score
			}
		}
	}

	return out, nil
}

func SaveEmbeddingToDBb(ctx context.Context, chatCompletion *openai.ChatCompletion, vector []float64) error {
	// float64 -> float32
	vec32 := make([]float32, len(vector))
	for i, v := range vector {
		vec32[i] = float32(v)
	}

	vecBytes := make([]byte, 4*len(vec32))
	for i, v := range vec32 {
		bits := math.Float32bits(v)
		binary.LittleEndian.PutUint32(vecBytes[i*4:], bits)
	}

	response := chatCompletion.Choices[0].Message.Content

	_, err := redisClient.HSet(ctx, "cache:"+uuid.NewString(), map[string]interface{}{
		"response":   response,
		"embedding":  vecBytes,
		"created_at": time.Now().Unix(),
	}).Result()

	return err

}
