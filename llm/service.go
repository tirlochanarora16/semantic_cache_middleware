package llm

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"

	"example.com/database"
	"github.com/openai/openai-go/v3"
)

type ClientService struct {
	client *openai.Client
}

type PromptResult struct {
	Response        string
	CacheHit        bool
	SimilarityScore float64
}

const similarityThreshold = 0.08

func NewClientService(client *openai.Client) *ClientService {
	return &ClientService{
		client: client,
	}
}

func (s *ClientService) ProcessPrompt(ctx context.Context, input string) (*PromptResult, error) {
	requestStartedAt := time.Now()

	vector, err := s.GenerateInputEmbeddings(ctx, input)

	if err != nil {
		return nil, fmt.Errorf("generate embedding: %w", err)
	}

	// search redis for similarity search
	searchResp, err := s.SimilaritySearch(ctx, vector)

	if err != nil {
		return nil, fmt.Errorf("serach cache error: %w", err)
	}

	if searchResp.Found && searchResp.Score <= similarityThreshold {
		log.Printf("cache hit distance=%f total_latency_ms=%d", searchResp.Score, time.Since(requestStartedAt).Milliseconds())
		return &PromptResult{
			Response:        searchResp.Response,
			CacheHit:        true,
			SimilarityScore: searchResp.Score,
		}, nil
	}

	if searchResp.Found {
		log.Printf("cache miss nearest_distance=%f", searchResp.Score)
	} else {
		log.Printf("cache miss nearest_distance=none")
	}

	startedAt := time.Now()

	// hit OpenAI to get the response
	completion, err := s.GetAiResponse(ctx, input)

	if err != nil {
		return nil, fmt.Errorf("error getting AI response %w", err)
	}

	response := completion.Choices[0].Message.Content
	latencyMs := int32(time.Since(startedAt).Milliseconds())

	if err := database.SaveResponseToPostgres(ctx, completion, input, latencyMs); err != nil {
		return nil, fmt.Errorf("save response to postgres %w", err)
	}

	if err := database.SaveEmbeddingToRedis(ctx, completion, vector, input); err != nil {
		return nil, fmt.Errorf("save response to redis: %w", err)
	}

	log.Printf("llm response saved total_latency_ms=%d llm_latency_ms=%d", time.Since(requestStartedAt).Milliseconds(), latencyMs)

	return &PromptResult{
		Response: response,
		CacheHit: false,
	}, nil
}

func (s *ClientService) GenerateInputEmbeddings(ctx context.Context, input string) ([]float64, error) {
	resp, err := s.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModelTextEmbedding3Small,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(input),
		},
	})

	if err != nil {
		return []float64{}, err
	}

	return resp.Data[0].Embedding, err
}

func (s *ClientService) SimilaritySearch(ctx context.Context, vector []float64) (*database.VectorSearchResult, error) {
	// float64 -> float32 -> []byte
	convertedResponse := make([]float32, len(vector))

	for i, v := range vector {
		convertedResponse[i] = float32(v)
	}

	vectorBytes := make([]byte, 4*len(convertedResponse))
	for i, v := range convertedResponse {
		binary.LittleEndian.PutUint32(vectorBytes[i*4:], math.Float32bits(v))
	}

	result, err := database.DoVectorSearch(ctx, vectorBytes)

	if err != nil {
		return &database.VectorSearchResult{}, err
	}

	return result, nil

}

func (s *ClientService) GetAiResponse(ctx context.Context, input string) (*openai.ChatCompletion, error) {
	chatCompletion, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(input),
		},
		Model: openai.ChatModelGPT4_1Mini,
	})

	if err != nil {
		return nil, err
	}

	return chatCompletion, nil
}
