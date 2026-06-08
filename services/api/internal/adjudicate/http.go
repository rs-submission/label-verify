package adjudicate

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ttb/labelverify/internal/verify"
)

type HTTPClient struct {
	baseURL string
	http    *http.Client
}

func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *HTTPClient) Adjudicate(ctx context.Context, req Request) (Decision, error) {
	if c.baseURL == "" {
		return Decision{}, fmt.Errorf("judge base URL is required")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return Decision{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/adjudicate", bytes.NewReader(body))
	if err != nil {
		return Decision{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return Decision{}, requestError("judge", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return Decision{}, judgeError("judge", resp.StatusCode, resp.Status, detail)
	}
	var decision Decision
	if err := json.NewDecoder(resp.Body).Decode(&decision); err != nil {
		return Decision{}, err
	}
	return decision, nil
}

type labelReadRequest struct {
	ImageBase64 string              `json:"image_base64"`
	MimeType    string              `json:"mime_type"`
	Application applicationEvidence `json:"application"`
}

type applicationEvidence struct {
	Brand             string `json:"brand"`
	ClassType         string `json:"class_type"`
	NetContents       string `json:"net_contents"`
	ABV               string `json:"abv"`
	GovernmentWarning string `json:"government_warning"`
	NameAndAddress    string `json:"name_and_address"`
}

type LabelRead struct {
	Brand             string `json:"brand"`
	ClassType         string `json:"class_type"`
	NetContents       string `json:"net_contents"`
	AlcoholContents   string `json:"alcohol_contents"`
	NameAndAddress    string `json:"name_and_address"`
	GovernmentWarning string `json:"government_warning"`
}

func (c *HTTPClient) ReadLabel(ctx context.Context, app verify.Application, image []byte) (LabelRead, error) {
	if c.baseURL == "" {
		return LabelRead{}, fmt.Errorf("judge base URL is required")
	}
	req := labelReadRequest{
		ImageBase64: base64.StdEncoding.EncodeToString(image),
		MimeType:    http.DetectContentType(image),
		Application: applicationEvidence{
			Brand:             app.Brand,
			ClassType:         app.ClassType,
			NetContents:       app.NetContents,
			ABV:               app.ABV,
			GovernmentWarning: app.GovernmentWarning,
			NameAndAddress:    app.NameAndAddress,
		},
	}
	body, err := json.Marshal(req)
	if err != nil {
		return LabelRead{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/read-label", bytes.NewReader(body))
	if err != nil {
		return LabelRead{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return LabelRead{}, requestError("judge label reader", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return LabelRead{}, judgeError("judge label reader", resp.StatusCode, resp.Status, detail)
	}
	var read LabelRead
	if err := json.NewDecoder(resp.Body).Decode(&read); err != nil {
		return LabelRead{}, err
	}
	return read, nil
}

func judgeError(label string, statusCode int, status string, detail []byte) error {
	message := strings.TrimSpace(string(detail))
	if statusCode == http.StatusTooManyRequests || isQuotaError(message) {
		return fmt.Errorf("%s rate limited by Vertex AI quota; try again later or disable AI Label Reader", label)
	}
	if isInvalidOutputError(message) {
		return fmt.Errorf("%s returned unreadable structured output; deterministic verification was used", label)
	}
	if message == "" {
		return fmt.Errorf("%s returned %s", label, status)
	}
	return fmt.Errorf("%s returned %s: %s", label, status, message)
}

func requestError(label string, err error) error {
	if isTimeoutError(err) {
		return fmt.Errorf("%s timed out; try again later or disable AI Label Reader", label)
	}
	return fmt.Errorf("%s unavailable: %w", label, err)
}

func isQuotaError(message string) bool {
	normalized := strings.ToLower(message)
	return strings.Contains(normalized, "resource_exhausted") ||
		strings.Contains(normalized, "resource exhausted") ||
		strings.Contains(normalized, "quota exhausted")
}

func isInvalidOutputError(message string) bool {
	normalized := strings.ToLower(message)
	return strings.Contains(normalized, "invalid label json") ||
		strings.Contains(normalized, "invalid json") ||
		strings.Contains(normalized, "json_invalid")
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	type timeout interface {
		Timeout() bool
	}
	var timeoutErr timeout
	return errors.As(err, &timeoutErr) && timeoutErr.Timeout()
}
