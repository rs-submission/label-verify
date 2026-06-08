package verify

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ttb/labelverify/internal/match"
	"github.com/ttb/labelverify/internal/ocr"
)

func TestVerifySingleConsistent(t *testing.T) {
	store := &fakeStore{app: sampleApplication()}
	ocrClient := &fakeOCR{response: ocr.RecognizeResponse{Regions: sampleRegions()}}
	service := NewService(store, ocrClient)

	verdict, err := service.VerifySingle(context.Background(), "app-1", []byte("image"))
	if err != nil {
		t.Fatalf("VerifySingle returned error: %v", err)
	}

	if verdict.Status != "consistent" {
		t.Fatalf("status=%q want consistent; fields=%+v", verdict.Status, verdict.Fields)
	}
	if !store.saved {
		t.Fatal("verification result was not persisted")
	}
	if len(ocrClient.langs) != 2 || ocrClient.langs[0] != "en" || ocrClient.langs[1] != "fr" {
		t.Fatalf("langs=%v want [en fr]", ocrClient.langs)
	}
}

func TestVerifySingleFlaggedOnWarningCaseMismatch(t *testing.T) {
	store := &fakeStore{app: sampleApplication()}
	regions := sampleRegions()
	regions[4].Text = "Government Warning"
	ocrClient := &fakeOCR{response: ocr.RecognizeResponse{Regions: regions}}
	service := NewService(store, ocrClient)

	verdict, err := service.VerifySingle(context.Background(), "app-1", []byte("image"))
	if err != nil {
		t.Fatalf("VerifySingle returned error: %v", err)
	}

	if verdict.Status != "flagged" {
		t.Fatalf("status=%q want flagged", verdict.Status)
	}
}

func TestVerifySingleAppliesAdjudicatorBeforeAggregate(t *testing.T) {
	store := &fakeStore{app: sampleApplication()}
	regions := sampleRegions()
	regions[1].Text = "RED TABLE WINE"
	ocrClient := &fakeOCR{response: ocr.RecognizeResponse{Regions: regions}}
	adjudicator := &fakeAdjudicator{}
	service := NewServiceWithAdjudicator(store, ocrClient, adjudicator)

	verdict, err := service.VerifySingle(context.Background(), "app-1", []byte("image"))
	if err != nil {
		t.Fatalf("VerifySingle returned error: %v", err)
	}

	if !adjudicator.called {
		t.Fatal("adjudicator was not called")
	}
	if verdict.Status != "consistent" {
		t.Fatalf("status=%q want consistent; fields=%+v", verdict.Status, verdict.Fields)
	}
}

func TestVerifySingleAppliesImageReviewerBeforeAggregate(t *testing.T) {
	store := &fakeStore{app: sampleApplication()}
	regions := sampleRegions()
	regions[1].Text = "INGREDIENTS"
	ocrClient := &fakeOCR{response: ocr.RecognizeResponse{Regions: regions}}
	reviewer := &fakeImageReviewer{}
	service := NewService(store, ocrClient)

	verdict, err := service.VerifySingleWithImageReviewer(context.Background(), "app-1", []byte("image"), reviewer)
	if err != nil {
		t.Fatalf("VerifySingleWithImageReviewer returned error: %v", err)
	}

	if !reviewer.started || !reviewer.future.reviewed {
		t.Fatalf("image reviewer was not started/reviewed: %+v", reviewer)
	}
	if verdict.Status != "consistent" {
		t.Fatalf("status=%q want consistent; fields=%+v", verdict.Status, verdict.Fields)
	}
}

func TestVerifySingleDoesNotSpendPersistenceDeadlineOnImageReviewer(t *testing.T) {
	store := &fakeStore{app: sampleApplication(), failOnExpiredSaveContext: true}
	regions := sampleRegions()
	regions[1].Text = "INGREDIENTS"
	ocrClient := &fakeOCR{response: ocr.RecognizeResponse{Regions: regions}, delay: 20 * time.Millisecond}
	reviewer := &fakeImageReviewer{future: fakeImageReviewFuture{waitForContext: true}}
	service := NewService(store, ocrClient).WithTimeout(50 * time.Millisecond)

	verdict, err := service.VerifySingleWithImageReviewer(context.Background(), "app-1", []byte("image"), reviewer)
	if err != nil {
		t.Fatalf("VerifySingleWithImageReviewer returned error: %v", err)
	}
	if !store.saved {
		t.Fatal("verification result was not persisted")
	}
	if verdict.Status != "flagged" {
		t.Fatalf("status=%q want flagged; fields=%+v", verdict.Status, verdict.Fields)
	}
	if !reviewer.future.reviewed || !reviewer.future.contextExpired {
		t.Fatalf("review future did not observe reserved timeout: %+v", reviewer.future)
	}
}

func TestVerifySingleStoresImageRefWhenImageStoreConfigured(t *testing.T) {
	store := &fakeStore{app: sampleApplication()}
	ocrClient := &fakeOCR{response: ocr.RecognizeResponse{Regions: sampleRegions()}}
	images := &fakeImages{}
	service := NewService(store, ocrClient).WithImageStore(images)

	if _, err := service.VerifySingle(context.Background(), "app-1", []byte("image")); err != nil {
		t.Fatalf("VerifySingle returned error: %v", err)
	}

	if len(images.putIDs) != 1 {
		t.Fatalf("stored images=%v want one", images.putIDs)
	}
	if !strings.HasPrefix(images.putIDs[0], "app-1-") {
		t.Fatalf("image id=%q want app-1 prefix", images.putIDs[0])
	}
	if store.imageRef != images.putIDs[0] {
		t.Fatalf("saved image ref=%q want %q", store.imageRef, images.putIDs[0])
	}
}

func TestVerifyBatchRunsWithBoundedConcurrency(t *testing.T) {
	store := &fakeStore{app: sampleApplication()}
	ocrClient := &fakeOCR{response: ocr.RecognizeResponse{Regions: sampleRegions()}, delay: 25 * time.Millisecond}
	service := NewService(store, ocrClient)

	items := make([]BatchItem, 10)
	for i := range items {
		items[i] = BatchItem{ApplicationID: "app-1", Image: []byte("image")}
	}

	results := service.VerifyBatch(context.Background(), items, 3)

	if len(results) != 10 {
		t.Fatalf("len(results)=%d want 10", len(results))
	}
	for _, result := range results {
		if result.Error != nil {
			t.Fatalf("unexpected result error: %v", result.Error)
		}
	}
	if ocrClient.maxInFlight < 2 {
		t.Fatalf("maxInFlight=%d want at least 2", ocrClient.maxInFlight)
	}
	if ocrClient.maxInFlight > 3 {
		t.Fatalf("maxInFlight=%d exceeded bound 3", ocrClient.maxInFlight)
	}
}

func TestNumberedFieldSupportsMoreThanNineBlocks(t *testing.T) {
	if got := numberedField("foreign_text", 9); got != "foreign_text_10" {
		t.Fatalf("numberedField returned %q, want foreign_text_10", got)
	}
}

const fullTTBWarning = "GOVERNMENT WARNING: (1) According to the Surgeon General, women should not drink alcoholic beverages during pregnancy because of the risk of birth defects. (2) Consumption of alcoholic beverages impairs your ability to drive a car or operate machinery, and may cause health problems."

func fieldByName(fields []match.FieldResult, name string) (match.FieldResult, bool) {
	for _, f := range fields {
		if f.Field == name {
			return f, true
		}
	}
	return match.FieldResult{}, false
}

// The full mandated warning is set vertically in tiny print, so real OCR returns
// it with per-character noise. Word coverage must still accept the complete-but-noisy
// statement, where exact equality could not.
func TestVerifySingleAcceptsCompleteWarningWithOCRNoise(t *testing.T) {
	app := sampleApplication()
	app.GovernmentWarning = fullTTBWarning
	noisy := "GOVERNMENT WARNNG: (1) Accordng to the Surgeon Genral, wornen should not drink alcoholc beverages durng pregnancy because of the risk of birth defects. (2) Consumpton of alcoholic beverages impairs your abilty to drive a car or operate machnery, and may cause health problms."
	regions := append(sampleRegions(), ocr.Region{Text: noisy, Confidence: 0.97, BBox: [4]float64{0.6, 0.5, 0.3, 0.2}})

	store := &fakeStore{app: app}
	service := NewService(store, &fakeOCR{response: ocr.RecognizeResponse{Regions: regions}})

	verdict, err := service.VerifySingle(context.Background(), "app-1", []byte("image"))
	if err != nil {
		t.Fatalf("VerifySingle returned error: %v", err)
	}
	gw, ok := fieldByName(verdict.Fields, "government_warning")
	if !ok {
		t.Fatal("government_warning field missing from verdict")
	}
	if !gw.Pass {
		t.Fatalf("complete (noisy) warning should pass; score=%.3f diff=%q", gw.Score, gw.Diff)
	}
}

func sampleApplication() Application {
	return Application{
		ID:                "app-1",
		Brand:             "Stone's Throw",
		ClassType:         "Red Wine",
		NetContents:       "750 mL",
		ABV:               "13.5% ALC/VOL",
		GovernmentWarning: "GOVERNMENT WARNING",
		NameAndAddress:    "Stone Throw Winery Richmond, VA",
		ForeignBlocks: []ForeignBlock{{
			Text:               "Produit de France",
			EnglishTranslation: "Product of France",
			Language:           "fr",
		}},
	}
}

func sampleRegions() []ocr.Region {
	return []ocr.Region{
		{Text: "STONE'S THROW", Confidence: 0.99, BBox: [4]float64{0.1, 0.1, 0.2, 0.02}},
		{Text: "Red Wine", Confidence: 0.99, BBox: [4]float64{0.1, 0.2, 0.2, 0.02}},
		{Text: "750 ml", Confidence: 0.99, BBox: [4]float64{0.1, 0.3, 0.2, 0.02}},
		{Text: "13.5% ALC/VOL", Confidence: 0.99, BBox: [4]float64{0.1, 0.4, 0.2, 0.02}},
		{Text: "GOVERNMENT WARNING", Confidence: 0.99, BBox: [4]float64{0.1, 0.5, 0.2, 0.02}},
		{Text: "Stone Throw Winery", Confidence: 0.99, BBox: [4]float64{0.1, 0.6, 0.2, 0.02}},
		{Text: "Richmond, VA", Confidence: 0.99, BBox: [4]float64{0.1, 0.63, 0.2, 0.02}},
		{Text: "Produit de France", Confidence: 0.99, BBox: [4]float64{0.1, 0.7, 0.2, 0.02}},
	}
}

type fakeStore struct {
	app                      Application
	saved                    bool
	imageRef                 string
	failOnExpiredSaveContext bool
}

func (s *fakeStore) GetApplication(ctx context.Context, id string) (Application, error) {
	return s.app, nil
}

func (s *fakeStore) SaveVerification(ctx context.Context, result StoredVerification) error {
	if s.failOnExpiredSaveContext && ctx.Err() != nil {
		return ctx.Err()
	}
	s.saved = true
	s.imageRef = result.ImageRef
	return nil
}

type fakeImages struct {
	putIDs []string
}

func (f *fakeImages) Put(id string, data []byte) (string, error) {
	f.putIDs = append(f.putIDs, id)
	return id, nil
}

type fakeOCR struct {
	response    ocr.RecognizeResponse
	langs       []string
	delay       time.Duration
	mu          sync.Mutex
	inFlight    int
	maxInFlight int
}

func (c *fakeOCR) Recognize(ctx context.Context, image []byte, langs []string) (ocr.RecognizeResponse, error) {
	c.mu.Lock()
	c.langs = langs
	c.inFlight++
	if c.inFlight > c.maxInFlight {
		c.maxInFlight = c.inFlight
	}
	c.mu.Unlock()

	if c.delay > 0 {
		time.Sleep(c.delay)
	}

	c.mu.Lock()
	c.inFlight--
	c.mu.Unlock()
	return c.response, nil
}

type fakeAdjudicator struct {
	called bool
}

func (a *fakeAdjudicator) ReviewFields(ctx context.Context, fields []match.FieldResult, pool []match.TextRegion) []match.FieldResult {
	a.called = true
	for i := range fields {
		if fields[i].Field == "class_type" {
			fields[i].Pass = true
			fields[i].Score = 0.96
			fields[i].Diff = ""
			fields[i].ReviewSource = "llm"
		}
	}
	return fields
}

type fakeImageReviewer struct {
	started bool
	future  fakeImageReviewFuture
}

func (r *fakeImageReviewer) StartReview(ctx context.Context, app Application, image []byte) ImageReviewFuture {
	r.started = true
	return &r.future
}

type fakeImageReviewFuture struct {
	reviewed       bool
	waitForContext bool
	contextExpired bool
}

func (f *fakeImageReviewFuture) ReviewFields(ctx context.Context, fields []match.FieldResult) []match.FieldResult {
	f.reviewed = true
	if f.waitForContext {
		<-ctx.Done()
		f.contextExpired = true
		return fields
	}
	for i := range fields {
		if fields[i].Field == "class_type" {
			fields[i].Pass = true
			fields[i].Score = 1
			fields[i].Diff = ""
			fields[i].ReviewSource = "ai_reader"
		}
	}
	return fields
}
