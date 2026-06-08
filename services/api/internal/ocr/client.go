package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Region struct {
	Text       string     `json:"text"`
	Confidence float64    `json:"confidence"`
	BBox       [4]float64 `json:"bbox"`
}

type RecognizeResponse struct {
	Regions   []Region `json:"regions"`
	ElapsedMS int64    `json:"elapsed_ms"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *Client) Recognize(ctx context.Context, image []byte, langs []string) (RecognizeResponse, error) {
	if c.baseURL == "" {
		return RecognizeResponse{}, fmt.Errorf("ocr base URL is required")
	}

	endpoint, err := url.Parse(c.baseURL + "/recognize")
	if err != nil {
		return RecognizeResponse{}, err
	}
	query := endpoint.Query()
	if len(langs) == 0 {
		langs = []string{"en"}
	}
	query.Set("langs", strings.Join(langs, ","))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(image))
	if err != nil {
		return RecognizeResponse{}, err
	}
	req.Header.Set("Content-Type", sniffImageContentType(image))

	resp, err := c.http.Do(req)
	if err != nil {
		return RecognizeResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RecognizeResponse{}, fmt.Errorf("ocr returned %s", resp.Status)
	}

	var out RecognizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return RecognizeResponse{}, err
	}
	return out, nil
}

func sniffImageContentType(image []byte) string {
	contentType := http.DetectContentType(image)
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
		return contentType
	default:
		return "application/octet-stream"
	}
}
