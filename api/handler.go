package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"example.com/llm"
)

type Handler struct {
	service *llm.ClientService
}

func NewHandler(service *llm.ClientService) *Handler {
	return &Handler{
		service: service,
	}
}

type generateRequest struct {
	Prompt string `json:"prompt"`
}

type generateResponse struct {
	Response        string  `json:"response"`
	CacheHit        bool    `json:"cache_hit"`
	SimilarityScore float64 `json:"similarity_score"`
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	var request generateRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	request.Prompt = strings.TrimSpace(request.Prompt)

	if request.Prompt == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	result, err := h.service.ProcessPrompt(ctx, request.Prompt)
	if err != nil {
		log.Printf("generate request failed: %v", err)
		http.Error(w, "unable to generate response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(generateResponse{
		Response:        result.Response,
		CacheHit:        result.CacheHit,
		SimilarityScore: result.SimilarityScore,
	})

}
