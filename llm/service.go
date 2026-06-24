package llm

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"

	"example.com/database"
	"example.com/helpers"
	"github.com/openai/openai-go/v3"
)

type ClientService struct {
	client *openai.Client
}

const similarityThreshold = 0.08

func NewClientService(client *openai.Client) *ClientService {
	return &ClientService{
		client: client,
	}
}

func (s *ClientService) RunAllServices(ctx context.Context) {
	// get input from the user
	input, err := helpers.GetUserInput()

	if err != nil {
		log.Fatalf("Error connecting to Redis %v", err)
		return
	}

	// convert the input into embeddings
	vector, err := s.GenerateInputEmbeddings(ctx, input)

	if err != nil {
		log.Fatalf("Error generating vector embedding for the input %v", err)
		return
	}

	// search redis for similarity search
	searchResp, err := s.SimilaritySearch(ctx, vector)

	if err != nil {
		log.Fatalf("Error getting the response from Redis %v", err)
		return
	}

	if searchResp.Found && searchResp.Score <= similarityThreshold {
		// get the response from the Redis DB
		fmt.Println(searchResp.Response)
		return
	}

	startTime := time.Now()

	// hit OpenAI to get the response
	chatCompletion, err := s.GetAiResponse(ctx, input)

	if err != nil {
		log.Fatalf("Error getting the response from Open AI %v", err)
		return
	}

	err = database.SaveEmbeddingToDBb(ctx, chatCompletion, vector, input)

	if err != nil {
		log.Fatalf("Error storing the response to Redis %v", err)
		return
	}

	latencyMs := int32(time.Since(startTime).Microseconds())

	err = database.SaveResponseToPostgres(ctx, chatCompletion, input, latencyMs)

	if err != nil {
		log.Fatalf("Error storing the response to Postgres %v", err)
		return
	}

	fmt.Println(chatCompletion.Choices[0].Message.Content)

	// save the complete data to postgres and redis
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
