package adjudicate

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestOllamaClientAdjudicate(t *testing.T) {
	var seen ollamaChatRequest
	client := NewOllamaClient("http://ollama.test", "gemma4:latest", time.Second)
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("path=%s want /api/chat", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&seen); err != nil {
			t.Fatal(err)
		}
		body, err := json.Marshal(ollamaChatResponse{
			Message: ollamaMessage{Content: `{"decision":"equivalent","confidence":0.86,"matched_text":"POM CHERRY CORDIAL","reason":"same class"}`},
		})
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	decision, err := client.Adjudicate(context.Background(), Request{
		Field:              "class_type",
		Expected:           "Cherry Liqueur/Cordial",
		Extracted:          "POM CHERRY CORDIAL",
		DeterministicScore: 0.67,
		OCRCandidates:      []string{"POM CHERRY CORDIAL"},
	})
	if err != nil {
		t.Fatalf("Adjudicate returned error: %v", err)
	}

	if seen.Model != "gemma4:latest" || seen.Format != "json" || seen.Stream {
		t.Fatalf("request=%+v", seen)
	}
	if decision.Decision != "equivalent" || decision.Confidence != 0.86 {
		t.Fatalf("decision=%+v", decision)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
