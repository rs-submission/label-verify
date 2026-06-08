package apihttp

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ttb/labelverify/internal/adjudicate"
	"github.com/ttb/labelverify/internal/match"
	"github.com/ttb/labelverify/internal/verify"
	"nhooyr.io/websocket"
)

const (
	maxUploadBytes      = 10 * 1024 * 1024
	maxBatchRecords     = 100
	maxBatchUploadBytes = maxBatchRecords*maxUploadBytes + 2*1024*1024
	defaultBatchWorkers = 1
	maxBatchWorkers     = 8
)

//go:embed web/*
var webFiles embed.FS

type ApplicationStore interface {
	SaveApplication(ctx context.Context, app verify.Application) error
	DeleteApplication(ctx context.Context, id string) (verify.DeletedApplication, error)
	ListApplications(ctx context.Context) ([]verify.ApplicationSummary, error)
	GetApplication(ctx context.Context, id string) (verify.Application, error)
}

type Verifier interface {
	VerifySingle(ctx context.Context, appID string, image []byte) (match.Verdict, error)
}

type AdjudicatingVerifier interface {
	Verifier
	VerifySingleWithAdjudicator(ctx context.Context, appID string, image []byte, adjudicator verify.FieldAdjudicator) (match.Verdict, error)
}

type ImageReviewingVerifier interface {
	Verifier
	VerifySingleWithImageReviewer(ctx context.Context, appID string, image []byte, reviewer verify.ImageReviewer) (match.Verdict, error)
}

type ImageDeleter interface {
	Delete(ref string) error
}

type RouterOptions struct {
	Judge JudgeRuntime
}

type JudgeRuntime struct {
	DefaultEnabled bool
	Addr           string
	TimeoutMS      int
	Policy         adjudicate.Policy
}

func NewRouter(apps ApplicationStore, verifier Verifier) http.Handler {
	return NewRouterWithImageStore(apps, verifier, nil)
}

func NewRouterWithImageStore(apps ApplicationStore, verifier Verifier, images ImageDeleter) http.Handler {
	return NewRouterWithOptions(apps, verifier, images, RouterOptions{})
}

func NewRouterWithOptions(apps ApplicationStore, verifier Verifier, images ImageDeleter, opts RouterOptions) http.Handler {
	r := chi.NewRouter()
	mountWeb(r)
	r.Get("/api/applications", listApplications(apps))
	r.Get("/api/applications/{application_id}", getApplication(apps))
	r.Put("/api/applications/{application_id}", upsertApplication(apps))
	r.Delete("/api/applications/{application_id}", deleteApplication(apps, images))
	r.Post("/api/verify", verifySingle(verifier, opts.Judge))
	r.Post("/api/verify/batch", verifyBatch(verifier, opts.Judge))
	r.Get("/ws/progress/{job_id}", progressStub)
	return r
}

func mountWeb(r chi.Router) {
	webRoot, err := fs.Sub(webFiles, "web")
	if err != nil {
		return
	}
	// Assets are embedded with a zero modtime, so the file server emits no
	// Last-Modified/ETag and browsers heuristically cache them. In this
	// prototype that serves a stale app.js against a fresh index.html, so
	// disable caching for the operator UI.
	static := noCache(http.FileServer(http.FS(webRoot)))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFileFS(w, r, webRoot, "index.html")
	})
	r.Handle("/static/*", http.StripPrefix("/static/", static))
}

func listApplications(apps ApplicationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summaries, err := apps.ListApplications(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"applications": summaries})
	}
}

func getApplication(apps ApplicationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID := strings.TrimSpace(chi.URLParam(r, "application_id"))
		if appID == "" {
			http.Error(w, "application_id is required", http.StatusBadRequest)
			return
		}
		app, err := apps.GetApplication(r.Context(), appID)
		if errors.Is(err, verify.ErrApplicationNotFound) {
			http.Error(w, "application not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, app)
	}
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func upsertApplication(apps ApplicationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID := strings.TrimSpace(chi.URLParam(r, "application_id"))
		if appID == "" {
			http.Error(w, "application_id is required", http.StatusBadRequest)
			return
		}

		var app verify.Application
		if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
			http.Error(w, "invalid application JSON", http.StatusBadRequest)
			return
		}
		if app.ID != "" && app.ID != appID {
			http.Error(w, "application ID in body must match URL", http.StatusBadRequest)
			return
		}
		app.ID = appID
		applyApplicationDefaults(&app)
		if err := apps.SaveApplication(r.Context(), app); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"application_id": app.ID})
	}
}

func applyApplicationDefaults(app *verify.Application) {
	if len(app.DeclaredLanguages) == 0 {
		app.DeclaredLanguages = []string{"en"}
	}
}

func deleteApplication(apps ApplicationStore, images ImageDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID := strings.TrimSpace(chi.URLParam(r, "application_id"))
		if appID == "" {
			http.Error(w, "application_id is required", http.StatusBadRequest)
			return
		}

		deleted, err := apps.DeleteApplication(r.Context(), appID)
		if errors.Is(err, verify.ErrApplicationNotFound) {
			http.Error(w, "application not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		deletedRefs := make([]string, 0, len(deleted.ImageRefs))
		failedRefs := make([]map[string]string, 0)
		for _, ref := range deleted.ImageRefs {
			if images != nil {
				if err := images.Delete(ref); err != nil {
					failedRefs = append(failedRefs, map[string]string{
						"ref":   ref,
						"error": err.Error(),
					})
					continue
				}
			}
			deletedRefs = append(deletedRefs, ref)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"application_id":     deleted.ApplicationID,
			"deleted_image_refs": deletedRefs,
			"failed_image_refs":  failedRefs,
		})
	}
}

func verifySingle(verifier Verifier, judge JudgeRuntime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
		if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
			http.Error(w, "invalid upload", http.StatusBadRequest)
			return
		}

		appID := r.FormValue("application_id")
		if appID == "" {
			http.Error(w, "application_id is required", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "image is required", http.StatusBadRequest)
			return
		}
		defer file.Close()
		if !allowedImageName(header.Filename) {
			http.Error(w, "unsupported image type", http.StatusBadRequest)
			return
		}
		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "could not read image", http.StatusBadRequest)
			return
		}

		imageReviewer, err := judge.imageReviewerFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var verdict match.Verdict
		if imageReviewer != nil {
			reviewing, ok := verifier.(ImageReviewingVerifier)
			if !ok {
				http.Error(w, "judge review is not available", http.StatusInternalServerError)
				return
			}
			verdict, err = reviewing.VerifySingleWithImageReviewer(r.Context(), appID, data, imageReviewer)
		} else {
			verdict, err = verifier.VerifySingle(r.Context(), appID, data)
		}
		if err != nil {
			writeVerificationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, verdict)
	}
}

func writeVerificationError(w http.ResponseWriter, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		http.Error(w, "verification timed out", http.StatusGatewayTimeout)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

type batchItemRequest struct {
	ID            string `json:"id"`
	ApplicationID string `json:"application_id"`
	ImageField    string `json:"image_field"`
}

type batchResponse struct {
	Status     string              `json:"status"`
	Count      int                 `json:"count"`
	Limit      int                 `json:"limit"`
	DurationMS int64               `json:"duration_ms"`
	Summary    batchSummary        `json:"summary"`
	Results    []batchResultRecord `json:"results"`
}

type batchSummary struct {
	Consistent int `json:"consistent"`
	Flagged    int `json:"flagged"`
	Errors     int `json:"errors"`
}

type batchResultRecord struct {
	ID            string         `json:"id"`
	ApplicationID string         `json:"application_id"`
	Status        string         `json:"status"`
	DurationMS    int64          `json:"duration_ms"`
	Verdict       *match.Verdict `json:"verdict,omitempty"`
	Error         string         `json:"error,omitempty"`
}

func verifyBatch(verifier Verifier, judge JudgeRuntime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		r.Body = http.MaxBytesReader(w, r.Body, maxBatchUploadBytes)
		if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
			http.Error(w, "invalid batch upload", http.StatusBadRequest)
			return
		}

		items, err := parseBatchItems(r.FormValue("items"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(items) == 0 {
			http.Error(w, "batch requires at least 1 item", http.StatusBadRequest)
			return
		}
		if len(items) > maxBatchRecords {
			http.Error(w, fmt.Sprintf("batch limit is %d records", maxBatchRecords), http.StatusBadRequest)
			return
		}

		imageReviewer, err := judge.imageReviewerFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		reviewing, ok := verifier.(ImageReviewingVerifier)
		if imageReviewer != nil && !ok {
			http.Error(w, "judge review is not available", http.StatusInternalServerError)
			return
		}

		images, err := readBatchImages(r, items)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		workers := boundedBatchWorkers(formInt(r, "max_concurrency", defaultBatchWorkers))
		results := runBatchVerifications(r.Context(), verifier, reviewing, imageReviewer, items, images, workers)
		response := batchResponse{
			Status:     "completed",
			Count:      len(items),
			Limit:      maxBatchRecords,
			DurationMS: time.Since(started).Milliseconds(),
			Results:    results,
		}
		for _, result := range results {
			switch result.Status {
			case "consistent":
				response.Summary.Consistent++
			case "flagged":
				response.Summary.Flagged++
			default:
				response.Summary.Errors++
			}
		}
		writeJSON(w, http.StatusOK, response)
	}
}

func parseBatchItems(value string) ([]batchItemRequest, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("items JSON is required")
	}
	var items []batchItemRequest
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil, fmt.Errorf("items must be a JSON array")
	}
	seenIDs := make(map[string]bool, len(items))
	seenImageFields := make(map[string]bool, len(items))
	for i := range items {
		items[i].ID = strings.TrimSpace(items[i].ID)
		items[i].ApplicationID = strings.TrimSpace(items[i].ApplicationID)
		items[i].ImageField = strings.TrimSpace(items[i].ImageField)
		if items[i].ID == "" {
			items[i].ID = fmt.Sprintf("row-%d", i+1)
		}
		if items[i].ApplicationID == "" {
			return nil, fmt.Errorf("items[%d].application_id is required", i)
		}
		if items[i].ImageField == "" {
			return nil, fmt.Errorf("items[%d].image_field is required", i)
		}
		if seenIDs[items[i].ID] {
			return nil, fmt.Errorf("duplicate item id %q", items[i].ID)
		}
		if seenImageFields[items[i].ImageField] {
			return nil, fmt.Errorf("duplicate image_field %q", items[i].ImageField)
		}
		seenIDs[items[i].ID] = true
		seenImageFields[items[i].ImageField] = true
	}
	return items, nil
}

func readBatchImages(r *http.Request, items []batchItemRequest) ([][]byte, error) {
	images := make([][]byte, len(items))
	for i, item := range items {
		file, header, err := r.FormFile(item.ImageField)
		if err != nil {
			return nil, fmt.Errorf("image field %q is required", item.ImageField)
		}
		if !allowedImageName(header.Filename) {
			_ = file.Close()
			return nil, fmt.Errorf("unsupported image type for %q", item.ImageField)
		}
		data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
		_ = file.Close()
		if err != nil {
			return nil, fmt.Errorf("could not read image field %q", item.ImageField)
		}
		if len(data) > maxUploadBytes {
			return nil, fmt.Errorf("image field %q exceeds %d bytes", item.ImageField, maxUploadBytes)
		}
		images[i] = data
	}
	return images, nil
}

func runBatchVerifications(ctx context.Context, verifier Verifier, reviewing ImageReviewingVerifier, imageReviewer verify.ImageReviewer, items []batchItemRequest, images [][]byte, workers int) []batchResultRecord {
	results := make([]batchResultRecord, len(items))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, item := range items {
		i, item := i, item
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[i] = batchResultRecord{ID: item.ID, ApplicationID: item.ApplicationID, Status: "error", Error: ctx.Err().Error()}
				return
			}

			started := time.Now()
			var verdict match.Verdict
			var err error
			if imageReviewer != nil {
				verdict, err = reviewing.VerifySingleWithImageReviewer(ctx, item.ApplicationID, images[i], imageReviewer)
			} else {
				verdict, err = verifier.VerifySingle(ctx, item.ApplicationID, images[i])
			}
			result := batchResultRecord{
				ID:            item.ID,
				ApplicationID: item.ApplicationID,
				DurationMS:    time.Since(started).Milliseconds(),
			}
			if err != nil {
				result.Status = "error"
				result.Error = err.Error()
			} else {
				result.Status = verdict.Status
				result.Verdict = &verdict
			}
			results[i] = result
		}()
	}
	wg.Wait()
	return results
}

func boundedBatchWorkers(value int) int {
	if value <= 0 {
		return defaultBatchWorkers
	}
	if value > maxBatchWorkers {
		return maxBatchWorkers
	}
	return value
}

func (j JudgeRuntime) adjudicatorFromRequest(r *http.Request) (verify.FieldAdjudicator, error) {
	policy, timeoutMS, enabled, err := j.policyFromRequest(r)
	if err != nil || !enabled {
		return nil, err
	}
	client := adjudicate.NewHTTPClient(j.Addr, time.Duration(timeoutMS)*time.Millisecond)
	return adjudicate.NewService(policy, client), nil
}

func (j JudgeRuntime) imageReviewerFromRequest(r *http.Request) (verify.ImageReviewer, error) {
	policy, timeoutMS, enabled, err := j.policyFromRequest(r)
	if err != nil || !enabled {
		return nil, err
	}
	client := adjudicate.NewHTTPClient(j.Addr, time.Duration(timeoutMS)*time.Millisecond)
	return adjudicate.NewImageReviewer(policy, client), nil
}

func (j JudgeRuntime) policyFromRequest(r *http.Request) (adjudicate.Policy, int, bool, error) {
	enabled := j.DefaultEnabled
	if value := strings.TrimSpace(r.FormValue("judge_enabled")); value != "" {
		parsed, err := parseBool(value)
		if err != nil {
			return adjudicate.Policy{}, 0, false, fmt.Errorf("judge_enabled must be true or false")
		}
		enabled = parsed
	}
	if !enabled {
		return adjudicate.Policy{}, 0, false, nil
	}
	if strings.TrimSpace(j.Addr) == "" {
		return adjudicate.Policy{}, 0, false, fmt.Errorf("judge service is not configured")
	}

	policy := j.Policy
	if policy.AllowedFields == nil {
		policy.AllowedFields = map[string]bool{"brand": true, "class_type": true}
	}
	if policy.DeniedFields == nil {
		policy.DeniedFields = map[string]bool{
			"abv":                true,
			"net_contents":       true,
			"government_warning": true,
			"name_address":       true,
		}
	}
	if allowedFields := strings.TrimSpace(r.FormValue("judge_allowed_fields")); allowedFields != "" {
		parsed, err := parseJudgeAllowedFields(allowedFields)
		if err != nil {
			return adjudicate.Policy{}, 0, false, err
		}
		policy.AllowedFields = parsed
		policy.DeniedFields = map[string]bool{}
	}
	policy.Enabled = true
	policy.Mode = formString(r, "judge_mode", defaultMode(policy.Mode))
	if policy.Mode != "shadow" && policy.Mode != "override" {
		return adjudicate.Policy{}, 0, false, fmt.Errorf("judge_mode must be shadow or override")
	}
	policy.MinDeterministicScore = 0
	policy.MaxDeterministicScore = 1
	if policy.MinLLMConfidence <= 0 {
		policy.MinLLMConfidence = 0.75
	}
	if policy.MinEligibleFailingFields <= 0 {
		policy.MinEligibleFailingFields = 1
	}

	timeoutMS := formInt(r, "judge_timeout_ms", defaultInt(j.TimeoutMS, 4500))
	if timeoutMS <= 0 {
		return adjudicate.Policy{}, 0, false, fmt.Errorf("judge_timeout_ms must be > 0")
	}
	return policy, timeoutMS, true, nil
}

func parseJudgeAllowedFields(value string) (map[string]bool, error) {
	known := map[string]bool{
		"brand":              true,
		"class_type":         true,
		"net_contents":       true,
		"abv":                true,
		"government_warning": true,
		"name_address":       true,
	}
	out := make(map[string]bool)
	for _, part := range strings.Split(value, ",") {
		field := strings.TrimSpace(part)
		if field == "" {
			continue
		}
		if !known[field] {
			return nil, fmt.Errorf("unknown judge field %q", field)
		}
		out[field] = true
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("judge_allowed_fields must include at least one field")
	}
	return out, nil
}

func progressStub(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")
	jobID := chi.URLParam(r, "job_id")
	_ = conn.Write(r.Context(), websocket.MessageText, []byte(fmt.Sprintf(`{"stage":"queued","current":0,"total":0,"message":"Job %s queued"}`, jobID)))
}

func allowedImageName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".jpg") ||
		strings.HasSuffix(lower, ".jpeg") ||
		strings.HasSuffix(lower, ".png") ||
		strings.HasSuffix(lower, ".webp")
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "on", "yes":
		return true, nil
	case "0", "false", "off", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool")
	}
}

func formString(r *http.Request, name, fallback string) string {
	value := strings.TrimSpace(r.FormValue(name))
	if value == "" {
		return fallback
	}
	return value
}

func formFloat(r *http.Request, name string, fallback float64) float64 {
	value := strings.TrimSpace(r.FormValue(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func formInt(r *http.Request, name string, fallback int) int {
	value := strings.TrimSpace(r.FormValue(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func defaultMode(value string) string {
	if strings.TrimSpace(value) == "" {
		return "shadow"
	}
	return value
}

func defaultFloat(value, fallback float64) float64 {
	if value == 0 {
		return fallback
	}
	return value
}

func defaultInt(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
