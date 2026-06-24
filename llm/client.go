package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func NewClient() (*openai.Client, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")

	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("Invalid OpenAI key")
	}

	client := openai.NewClient(
		option.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	)

	return &client, nil
}

func CheckOpenAI(ctx context.Context, client *openai.Client) error {
	if client == nil {
		return fmt.Errorf("OpenAI client is not initalized")
	}

	if _, err := client.Models.List(ctx); err != nil {
		return fmt.Errorf("OpenAI startup check failed: %w", err)
	}

	return nil
}
