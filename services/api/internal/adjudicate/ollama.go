package adjudicate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OllamaClient struct {
	baseURL string
	model   string
	http    *http.Client
}

func NewOllamaClient(baseURL, model string, timeout time.Duration) *OllamaClient {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &OllamaClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *OllamaClient) Adjudicate(ctx context.Context, req Request) (Decision, error) {
	if c.baseURL == "" {
		return Decision{}, fmt.Errorf("llm base URL is required")
	}
	if c.model == "" {
		return Decision{}, fmt.Errorf("llm model is required")
	}

	evidence, err := json.Marshal(req)
	if err != nil {
		return Decision{}, err
	}
	body, err := json.Marshal(ollamaChatRequest{
		Model:  c.model,
		Stream: false,
		Format: "json",
		Options: map[string]any{
			"temperature": 0,
		},
		Messages: []ollamaMessage{
			{Role: "system", Content: adjudicatorSystemPrompt},
			{Role: "user", Content: string(evidence)},
		},
	})
	if err != nil {
		return Decision{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return Decision{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return Decision{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Decision{}, fmt.Errorf("llm returned %s", resp.Status)
	}

	var chat ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chat); err != nil {
		return Decision{}, err
	}
	var decision Decision
	if err := json.Unmarshal([]byte(chat.Message.Content), &decision); err != nil {
		return Decision{}, err
	}
	return decision, nil
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   string          `json:"format"`
	Options  map[string]any  `json:"options"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

const adjudicatorSystemPrompt = `You adjudicate alcohol label verification fields. Return only JSON with keys decision, confidence, matched_text, reason.
Allowed decisions are equivalent, not_equivalent, uncertain.
Use the provided structured evidence only.
Mark equivalent only when the expected field and extracted OCR text mean the same label value.
Do not invent missing text.
Be conservative.`
