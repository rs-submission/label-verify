package apihttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/ttb/labelverify/internal/match"
	"github.com/ttb/labelverify/internal/verify"
)

func TestUpsertApplicationUsesPathID(t *testing.T) {
	apps := &fakeApps{}
	router := NewRouter(apps, &fakeVerifier{})

	body, _ := json.Marshal(verify.Application{Brand: "Stone's Throw"})
	req := httptest.NewRequest(http.MethodPut, "/api/applications/app-1", bytes.NewReader(body))
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", res.Code, res.Body.String())
	}
	if apps.saved.ID != "app-1" {
		t.Fatalf("saved app id=%q want app-1", apps.saved.ID)
	}
	if len(apps.saved.DeclaredLanguages) != 1 || apps.saved.DeclaredLanguages[0] != "en" {
		t.Fatalf("saved declared languages=%v want [en]", apps.saved.DeclaredLanguages)
	}
}

func TestUpsertApplicationRejectsConflictingBodyID(t *testing.T) {
	router := NewRouter(&fakeApps{}, &fakeVerifier{})

	body, _ := json.Marshal(verify.Application{ID: "other", Brand: "Stone's Throw"})
	req := httptest.NewRequest(http.MethodPut, "/api/applications/app-1", bytes.NewReader(body))
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", res.Code)
	}
}

func TestDeleteApplicationDeletesImageRefs(t *testing.T) {
	apps := &fakeApps{
		deleted: verify.DeletedApplication{
			ApplicationID: "app-1",
			ImageRefs:     []string{"app-1-image.jpg"},
		},
	}
	images := &fakeImages{}
	router := NewRouterWithImageStore(apps, &fakeVerifier{}, images)
	req := httptest.NewRequest(http.MethodDelete, "/api/applications/app-1", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", res.Code, res.Body.String())
	}
	if apps.deletedID != "app-1" {
		t.Fatalf("deleted id=%q want app-1", apps.deletedID)
	}
	if len(images.deleted) != 1 || images.deleted[0] != "app-1-image.jpg" {
		t.Fatalf("deleted images=%v want [app-1-image.jpg]", images.deleted)
	}
}

func TestDeleteApplicationReturnsNotFound(t *testing.T) {
	router := NewRouter(&fakeApps{deleteErr: verify.ErrApplicationNotFound}, &fakeVerifier{})
	req := httptest.NewRequest(http.MethodDelete, "/api/applications/missing", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", res.Code)
	}
}

func TestDeleteApplicationReportsImageDeleteFailures(t *testing.T) {
	apps := &fakeApps{
		deleted: verify.DeletedApplication{
			ApplicationID: "app-1",
			ImageRefs:     []string{"app-1-image.jpg"},
		},
	}
	images := &fakeImages{err: errors.New("cannot delete image")}
	router := NewRouterWithImageStore(apps, &fakeVerifier{}, images)
	req := httptest.NewRequest(http.MethodDelete, "/api/applications/app-1", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", res.Code, res.Body.String())
	}
	var body struct {
		DeletedImageRefs []string            `json:"deleted_image_refs"`
		FailedImageRefs  []map[string]string `json:"failed_image_refs"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.DeletedImageRefs) != 0 {
		t.Fatalf("deleted refs=%v want none", body.DeletedImageRefs)
	}
	if len(body.FailedImageRefs) != 1 || body.FailedImageRefs[0]["ref"] != "app-1-image.jpg" {
		t.Fatalf("failed refs=%v want app-1-image.jpg", body.FailedImageRefs)
	}
}

func TestListApplicationsReturnsSummaries(t *testing.T) {
	apps := &fakeApps{list: []verify.ApplicationSummary{
		{ID: "app-1", Brand: "POM VODKA", ClassType: "Vodka"},
		{ID: "app-2", Brand: "POM BOURBON", ClassType: "Bourbon"},
	}}
	router := NewRouter(apps, &fakeVerifier{})
	req := httptest.NewRequest(http.MethodGet, "/api/applications", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", res.Code, res.Body.String())
	}
	var body struct {
		Applications []verify.ApplicationSummary `json:"applications"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Applications) != 2 || body.Applications[0].ID != "app-1" || body.Applications[1].Brand != "POM BOURBON" {
		t.Fatalf("applications=%+v want app-1, app-2 summaries", body.Applications)
	}
}

func TestGetApplicationReturnsDetails(t *testing.T) {
	apps := &fakeApps{got: verify.Application{ID: "app-1", Brand: "POM VODKA", ClassType: "Vodka", ABV: "40% Alc./Vol."}}
	router := NewRouter(apps, &fakeVerifier{})
	req := httptest.NewRequest(http.MethodGet, "/api/applications/app-1", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", res.Code, res.Body.String())
	}
	if apps.gotID != "app-1" {
		t.Fatalf("requested id=%q want app-1", apps.gotID)
	}
	var app verify.Application
	if err := json.NewDecoder(res.Body).Decode(&app); err != nil {
		t.Fatal(err)
	}
	if app.Brand != "POM VODKA" || app.ABV != "40% Alc./Vol." {
		t.Fatalf("app=%+v want full POM VODKA details", app)
	}
}

func TestGetApplicationReturnsNotFound(t *testing.T) {
	router := NewRouter(&fakeApps{getErr: verify.ErrApplicationNotFound}, &fakeVerifier{})
	req := httptest.NewRequest(http.MethodGet, "/api/applications/missing", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", res.Code)
	}
}

func TestWebRootServesApp(t *testing.T) {
	router := NewRouter(&fakeApps{}, &fakeVerifier{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", res.Code, res.Body.String())
	}
	if !bytes.Contains(res.Body.Bytes(), []byte("Label Verification")) {
		t.Fatalf("body does not contain app shell")
	}
}

func TestVerifySingleUpload(t *testing.T) {
	verifier := &fakeVerifier{verdict: match.Verdict{Status: "consistent", Confidence: 0.98}}
	router := NewRouter(&fakeApps{}, verifier)
	body, contentType := multipartUpload(t, "label.jpg", []byte("image bytes"))
	req := httptest.NewRequest(http.MethodPost, "/api/verify", body)
	req.Header.Set("Content-Type", contentType)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", res.Code, res.Body.String())
	}
	if verifier.appID != "app-1" {
		t.Fatalf("appID=%q want app-1", verifier.appID)
	}
	if string(verifier.image) != "image bytes" {
		t.Fatalf("image=%q want image bytes", string(verifier.image))
	}
}

func TestVerifySingleUploadCanEnableJudgePerRequest(t *testing.T) {
	verifier := &fakeVerifier{verdict: match.Verdict{Status: "consistent", Confidence: 0.98}}
	router := NewRouterWithOptions(&fakeApps{}, verifier, nil, RouterOptions{
		Judge: JudgeRuntime{
			Addr:      "http://judge.test",
			TimeoutMS: 500,
		},
	})
	body, contentType := multipartUploadWithFields(t, "label.jpg", []byte("image bytes"), map[string]string{
		"judge_enabled":    "true",
		"judge_mode":       "shadow",
		"judge_timeout_ms": "500",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/verify", body)
	req.Header.Set("Content-Type", contentType)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", res.Code, res.Body.String())
	}
	if !verifier.usedImageReviewer {
		t.Fatal("verify request did not use per-request image reviewer")
	}
}

func TestVerifyBatchUpload(t *testing.T) {
	verifier := &fakeVerifier{verdict: match.Verdict{Status: "consistent", Confidence: 0.98}}
	router := NewRouter(&fakeApps{}, verifier)
	items := []batchItemRequest{
		{ID: "row-1", ApplicationID: "app-1", ImageField: "image_1"},
		{ID: "row-2", ApplicationID: "app-2", ImageField: "image_2"},
	}
	body, contentType := multipartBatchUpload(t, items, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/verify/batch", body)
	req.Header.Set("Content-Type", contentType)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", res.Code, res.Body.String())
	}
	var response batchResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Count != 2 || response.Limit != 100 || response.Summary.Consistent != 2 || len(response.Results) != 2 {
		t.Fatalf("response=%+v want 2 consistent results", response)
	}
	if response.Results[0].ID != "row-1" || response.Results[1].ApplicationID != "app-2" {
		t.Fatalf("results=%+v not in manifest order", response.Results)
	}
}

func TestVerifyBatchRejectsOverLimit(t *testing.T) {
	router := NewRouter(&fakeApps{}, &fakeVerifier{})
	items := make([]batchItemRequest, 101)
	for i := range items {
		items[i] = batchItemRequest{
			ID:            "row-" + strconv.Itoa(i+1),
			ApplicationID: "app-" + strconv.Itoa(i+1),
			ImageField:    "image_" + strconv.Itoa(i+1),
		}
	}
	body, contentType := multipartBatchUpload(t, items, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/verify/batch", body)
	req.Header.Set("Content-Type", contentType)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", res.Code)
	}
}

func TestVerifyBatchCanEnableJudgePerRequest(t *testing.T) {
	verifier := &fakeVerifier{verdict: match.Verdict{Status: "consistent", Confidence: 0.98}}
	router := NewRouterWithOptions(&fakeApps{}, verifier, nil, RouterOptions{
		Judge: JudgeRuntime{Addr: "http://judge.test", TimeoutMS: 500},
	})
	items := []batchItemRequest{{ID: "row-1", ApplicationID: "app-1", ImageField: "image_1"}}
	body, contentType := multipartBatchUpload(t, items, map[string]string{
		"judge_enabled": "true",
		"judge_mode":    "shadow",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/verify/batch", body)
	req.Header.Set("Content-Type", contentType)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d want 200 body=%s", res.Code, res.Body.String())
	}
	if !verifier.usedImageReviewer {
		t.Fatal("batch verify request did not use per-request image reviewer")
	}
}

func TestParseJudgeAllowedFields(t *testing.T) {
	fields, err := parseJudgeAllowedFields("brand, government_warning, abv")
	if err != nil {
		t.Fatal(err)
	}
	if !fields["brand"] || !fields["government_warning"] || !fields["abv"] || fields["class_type"] {
		t.Fatalf("fields=%v", fields)
	}
}

func TestParseJudgeAllowedFieldsRejectsUnknown(t *testing.T) {
	if _, err := parseJudgeAllowedFields("brand,unknown"); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestVerifyRejectsUnsupportedExtension(t *testing.T) {
	router := NewRouter(&fakeApps{}, &fakeVerifier{})
	body, contentType := multipartUpload(t, "label.txt", []byte("image bytes"))
	req := httptest.NewRequest(http.MethodPost, "/api/verify", body)
	req.Header.Set("Content-Type", contentType)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", res.Code)
	}
}

func multipartUpload(t *testing.T, filename string, data []byte) (*bytes.Buffer, string) {
	return multipartUploadWithFields(t, filename, data, nil)
}

func multipartUploadWithFields(t *testing.T, filename string, data []byte, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("application_id", "app-1"); err != nil {
		t.Fatal(err)
	}
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body, writer.FormDataContentType()
}

func multipartBatchUpload(t *testing.T, items []batchItemRequest, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("items", string(itemsJSON)); err != nil {
		t.Fatal(err)
	}
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range items {
		part, err := writer.CreateFormFile(item.ImageField, item.ImageField+".jpg")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte("image bytes")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body, writer.FormDataContentType()
}

type fakeApps struct {
	saved     verify.Application
	deleted   verify.DeletedApplication
	deletedID string
	deleteErr error
	list      []verify.ApplicationSummary
	got       verify.Application
	gotID     string
	getErr    error
}

func (f *fakeApps) SaveApplication(ctx context.Context, app verify.Application) error {
	f.saved = app
	return nil
}

func (f *fakeApps) ListApplications(ctx context.Context) ([]verify.ApplicationSummary, error) {
	return f.list, nil
}

func (f *fakeApps) GetApplication(ctx context.Context, id string) (verify.Application, error) {
	f.gotID = id
	if f.getErr != nil {
		return verify.Application{}, f.getErr
	}
	return f.got, nil
}

func (f *fakeApps) DeleteApplication(ctx context.Context, id string) (verify.DeletedApplication, error) {
	f.deletedID = id
	if f.deleteErr != nil {
		return verify.DeletedApplication{}, f.deleteErr
	}
	if f.deleted.ApplicationID == "" {
		f.deleted.ApplicationID = id
	}
	return f.deleted, nil
}

type fakeImages struct {
	deleted []string
	err     error
}

func (f *fakeImages) Delete(ref string) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = append(f.deleted, ref)
	return nil
}

type fakeVerifier struct {
	verdict           match.Verdict
	appID             string
	image             []byte
	usedAdjudicator   bool
	usedImageReviewer bool
}

func (f *fakeVerifier) VerifySingle(ctx context.Context, appID string, image []byte) (match.Verdict, error) {
	f.appID = appID
	f.image = image
	if f.verdict.Status == "" {
		f.verdict = match.Verdict{Status: "consistent", Confidence: 1}
	}
	return f.verdict, nil
}

func (f *fakeVerifier) VerifySingleWithAdjudicator(ctx context.Context, appID string, image []byte, adjudicator verify.FieldAdjudicator) (match.Verdict, error) {
	f.usedAdjudicator = adjudicator != nil
	return f.VerifySingle(ctx, appID, image)
}

func (f *fakeVerifier) VerifySingleWithImageReviewer(ctx context.Context, appID string, image []byte, reviewer verify.ImageReviewer) (match.Verdict, error) {
	f.usedImageReviewer = reviewer != nil
	return f.VerifySingle(ctx, appID, image)
}
