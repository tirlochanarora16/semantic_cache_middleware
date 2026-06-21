package llm

import (
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
