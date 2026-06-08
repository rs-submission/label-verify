package adjudicate

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ttb/labelverify/internal/verify"
)

func TestHTTPClientAdjudicate(t *testing.T) {
	var seen Request
	client := NewHTTPClient("http://judge.test", time.Second)
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/adjudicate" {
			t.Fatalf("path=%s want /adjudicate", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&seen); err != nil {
			t.Fatal(err)
		}
		body, err := json.Marshal(Decision{
			Decision:    "equivalent",
			Confidence:  0.86,
			MatchedText: "POM CHERRY CORDIAL",
			Reason:      "same class",
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

	if seen.Field != "class_type" || seen.Expected != "Cherry Liqueur/Cordial" {
		t.Fatalf("request=%+v", seen)
	}
	if decision.Decision != "equivalent" || decision.Confidence != 0.86 {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestHTTPClientIncludesErrorBody(t *testing.T) {
	client := NewHTTPClient("http://judge.test", time.Second)
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Status:     "502 Bad Gateway",
			Body:       io.NopCloser(bytes.NewBufferString(`{"detail":"model not found"}`)),
			Header:     make(http.Header),
		}, nil
	})}

	_, err := client.Adjudicate(context.Background(), Request{Field: "brand"})
	if err == nil || !strings.Contains(err.Error(), "502 Bad Gateway") || !strings.Contains(err.Error(), "model not found") {
		t.Fatalf("error=%v want status and response body", err)
	}
}

func TestHTTPClientShortensQuotaErrors(t *testing.T) {
	client := NewHTTPClient("http://judge.test", time.Second)
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Status:     "429 Too Many Requests",
			Body:       io.NopCloser(bytes.NewBufferString(`{"detail":"429 RESOURCE_EXHAUSTED. Resource exhausted."}`)),
			Header:     make(http.Header),
		}, nil
	})}

	_, err := client.ReadLabel(context.Background(), verify.Application{}, []byte("image"))
	if err == nil || !strings.Contains(err.Error(), "rate limited by Vertex AI quota") {
		t.Fatalf("error=%v want short quota message", err)
	}
	if strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Fatalf("error=%v should hide raw provider body", err)
	}
}

func TestHTTPClientShortensTimeoutErrors(t *testing.T) {
	client := NewHTTPClient("http://judge.test", time.Second)
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, &url.Error{Op: "Post", URL: r.URL.String(), Err: context.DeadlineExceeded}
	})}

	_, err := client.ReadLabel(context.Background(), verify.Application{}, []byte("image"))
	if err == nil || !strings.Contains(err.Error(), "judge label reader timed out") {
		t.Fatalf("error=%v want short timeout message", err)
	}
	if strings.Contains(err.Error(), "Client.Timeout") || strings.Contains(err.Error(), "awaiting headers") {
		t.Fatalf("error=%v should hide raw transport timeout", err)
	}
}

func TestHTTPClientShortensInvalidJSONErrors(t *testing.T) {
	client := NewHTTPClient("http://judge.test", time.Second)
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Status:     "502 Bad Gateway",
			Body: io.NopCloser(bytes.NewBufferString(
				`{"detail":"judge returned invalid label JSON: Invalid JSON: EOF while parsing a value [type=json_invalid]"}`,
			)),
			Header: make(http.Header),
		}, nil
	})}

	_, err := client.ReadLabel(context.Background(), verify.Application{}, []byte("image"))
	if err == nil || !strings.Contains(err.Error(), "returned unreadable structured output") {
		t.Fatalf("error=%v want short invalid JSON message", err)
	}
	if strings.Contains(err.Error(), "json_invalid") || strings.Contains(err.Error(), "EOF while parsing") {
		t.Fatalf("error=%v should hide raw validation details", err)
	}
}

func TestHTTPClientReadLabel(t *testing.T) {
	var seen labelReadRequest
	client := NewHTTPClient("http://judge.test", time.Second)
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/read-label" {
			t.Fatalf("path=%s want /read-label", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&seen); err != nil {
			t.Fatal(err)
		}
		body, err := json.Marshal(LabelRead{
			Brand:             "POM BOURBON",
			ClassType:         "Bourbon",
			NetContents:       "750 mL",
			AlcoholContents:   "45% ALC/VOL (90 Proof)",
			NameAndAddress:    "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
			GovernmentWarning: "GOVERNMENT WARNING",
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

	read, err := client.ReadLabel(context.Background(), verify.Application{
		Brand:             "POM BOURBON",
		ClassType:         "Bourbon",
		NetContents:       "750 mL",
		ABV:               "45% ALC/VOL (90 Proof)",
		GovernmentWarning: "GOVERNMENT WARNING",
		NameAndAddress:    "POM CREEK DISTILLING COMPANY, LLC PURCELLVILLE, VA",
	}, []byte("image"))
	if err != nil {
		t.Fatalf("ReadLabel returned error: %v", err)
	}

	if seen.ImageBase64 != "aW1hZ2U=" || seen.Application.Brand != "POM BOURBON" {
		t.Fatalf("request=%+v", seen)
	}
	if read.ClassType != "Bourbon" || read.GovernmentWarning != "GOVERNMENT WARNING" {
		t.Fatalf("read=%+v", read)
	}
}
