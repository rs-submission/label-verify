package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ttb/labelverify/internal/verify"
)

const maxRecords = 100

type manifest struct {
	Items []manifestItem `json:"items"`
}

type manifestItem struct {
	ID            string              `json:"id"`
	ApplicationID string              `json:"application_id"`
	Image         string              `json:"image"`
	Application   *verify.Application `json:"application,omitempty"`
}

type batchItem struct {
	ID            string `json:"id"`
	ApplicationID string `json:"application_id"`
	ImageField    string `json:"image_field"`
}

func main() {
	manifestPath := flag.String("manifest", "", "path to batch manifest JSON")
	apiBase := flag.String("api", "http://localhost:8080", "API base URL")
	outPath := flag.String("out", "", "path to write JSON batch response; empty writes only to stdout")
	concurrency := flag.Int("concurrency", 1, "batch worker count; server caps this at 8")
	requestTimeout := flag.Duration("request-timeout", 3*time.Minute, "HTTP request timeout for the full batch")
	judgeEnabled := flag.Bool("judge-enabled", false, "enable AI judge review for eligible fields")
	judgeMode := flag.String("judge-mode", "shadow", "judge mode: shadow or override")
	judgeFields := flag.String("judge-fields", "brand,class_type", "comma-separated fields eligible for judge review")
	judgeTimeoutMS := flag.Int("judge-timeout-ms", 4500, "judge service timeout per review in milliseconds")
	flag.Parse()

	if strings.TrimSpace(*manifestPath) == "" {
		log.Fatal("--manifest is required")
	}
	items, err := loadManifest(*manifestPath)
	if err != nil {
		log.Fatal(err)
	}
	if len(items) == 0 {
		log.Fatal("manifest must contain at least 1 item")
	}
	if len(items) > maxRecords {
		log.Fatalf("manifest has %d records; limit is %d", len(items), maxRecords)
	}

	client := &http.Client{Timeout: *requestTimeout}
	base := strings.TrimRight(*apiBase, "/")
	if err := upsertManifestApplications(context.Background(), client, base, items); err != nil {
		log.Fatal(err)
	}

	response, err := postBatch(context.Background(), client, base, items, batchOptions{
		Concurrency:    *concurrency,
		JudgeEnabled:   *judgeEnabled,
		JudgeMode:      *judgeMode,
		JudgeFields:    *judgeFields,
		JudgeTimeoutMS: *judgeTimeoutMS,
	})
	if err != nil {
		log.Fatal(err)
	}
	response = prettyJSON(response)
	if strings.TrimSpace(*outPath) != "" {
		if err := os.WriteFile(*outPath, response, 0o600); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println(string(response))
}

func loadManifest(path string) ([]manifestItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var list []manifestItem
	if err := json.Unmarshal(data, &list); err == nil && list != nil {
		return normalizeItems(list, filepath.Dir(path))
	}
	var doc manifest
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("manifest must be a JSON array or object with an items array: %w", err)
	}
	return normalizeItems(doc.Items, filepath.Dir(path))
}

func normalizeItems(items []manifestItem, baseDir string) ([]manifestItem, error) {
	seen := make(map[string]bool, len(items))
	for i := range items {
		items[i].ID = strings.TrimSpace(items[i].ID)
		items[i].ApplicationID = strings.TrimSpace(items[i].ApplicationID)
		items[i].Image = strings.TrimSpace(items[i].Image)
		if items[i].ID == "" {
			items[i].ID = fmt.Sprintf("row-%d", i+1)
		}
		if items[i].ApplicationID == "" {
			return nil, fmt.Errorf("items[%d].application_id is required", i)
		}
		if items[i].Image == "" {
			return nil, fmt.Errorf("items[%d].image is required", i)
		}
		if !filepath.IsAbs(items[i].Image) {
			items[i].Image = filepath.Clean(filepath.Join(baseDir, items[i].Image))
		}
		if seen[items[i].ID] {
			return nil, fmt.Errorf("duplicate item id %q", items[i].ID)
		}
		seen[items[i].ID] = true
	}
	return items, nil
}

func upsertManifestApplications(ctx context.Context, client *http.Client, apiBase string, items []manifestItem) error {
	for _, item := range items {
		if item.Application == nil {
			continue
		}
		app := *item.Application
		if strings.TrimSpace(app.ID) == "" {
			app.ID = item.ApplicationID
		}
		if app.ID != item.ApplicationID {
			return fmt.Errorf("item %q application.ID must match application_id", item.ID)
		}
		body, err := json.Marshal(app)
		if err != nil {
			return err
		}
		endpoint := apiBase + "/api/applications/" + url.PathEscape(item.ApplicationID)
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("upsert application %q: %w", item.ApplicationID, err)
		}
		responseBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("upsert application %q returned %s: %s", item.ApplicationID, resp.Status, strings.TrimSpace(string(responseBody)))
		}
	}
	return nil
}

type batchOptions struct {
	Concurrency    int
	JudgeEnabled   bool
	JudgeMode      string
	JudgeFields    string
	JudgeTimeoutMS int
}

func postBatch(ctx context.Context, client *http.Client, apiBase string, items []manifestItem, opts batchOptions) ([]byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	batchItems := make([]batchItem, 0, len(items))
	for i, item := range items {
		batchItems = append(batchItems, batchItem{
			ID:            item.ID,
			ApplicationID: item.ApplicationID,
			ImageField:    "image_" + strconv.Itoa(i+1),
		})
	}
	itemsJSON, err := json.Marshal(batchItems)
	if err != nil {
		return nil, err
	}
	fields := map[string]string{
		"items":                string(itemsJSON),
		"max_concurrency":      strconv.Itoa(opts.Concurrency),
		"judge_enabled":        strconv.FormatBool(opts.JudgeEnabled),
		"judge_mode":           opts.JudgeMode,
		"judge_allowed_fields": opts.JudgeFields,
		"judge_timeout_ms":     strconv.Itoa(opts.JudgeTimeoutMS),
	}
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return nil, err
		}
	}
	for i, item := range items {
		if err := addImagePart(writer, "image_"+strconv.Itoa(i+1), item.Image); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(apiBase, "/") + "/api/verify/batch"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("batch request returned %s: %s", resp.Status, strings.TrimSpace(string(response)))
	}
	return response, nil
}

func addImagePart(writer *multipart.Writer, field, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	part, err := writer.CreateFormFile(field, filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	return nil
}

func prettyJSON(data []byte) []byte {
	var out bytes.Buffer
	if err := json.Indent(&out, data, "", "  "); err != nil {
		return data
	}
	out.WriteByte('\n')
	return out.Bytes()
}
