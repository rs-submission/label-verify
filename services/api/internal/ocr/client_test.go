package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestRecognizePostsRawImageAndDecodesResponse(t *testing.T) {
	var gotContentType string
	var gotLangs string
	client := NewClient("http://ocr.test", time.Second)
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotContentType = r.Header.Get("Content-Type")
		gotLangs = r.URL.Query().Get("langs")
		body, err := json.Marshal(RecognizeResponse{
			Regions: []Region{{
				Text:       "GOVERNMENT WARNING",
				Confidence: 0.97,
				BBox:       [4]float64{0.1, 0.62, 0.55, 0.08},
			}},
			ElapsedMS: 740,
		})
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(bytes.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}

	resp, err := client.Recognize(context.Background(), []byte{0xff, 0xd8, 0xff, 0x00}, []string{"en", "fr"})
	if err != nil {
		t.Fatalf("Recognize returned error: %v", err)
	}

	if gotContentType != "image/jpeg" {
		t.Fatalf("Content-Type=%q want image/jpeg", gotContentType)
	}
	if gotLangs != "en,fr" {
		t.Fatalf("langs=%q want en,fr", gotLangs)
	}
	if len(resp.Regions) != 1 || resp.Regions[0].Text != "GOVERNMENT WARNING" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRecognizeReturnsErrorForNonOK(t *testing.T) {
	client := NewClient("http://ocr.test", time.Second)
	client.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     "400 Bad Request",
			Body:       io.NopCloser(bytes.NewReader([]byte("bad image"))),
			Header:     make(http.Header),
		}, nil
	})}
	if _, err := client.Recognize(context.Background(), []byte("bad"), []string{"en"}); err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
