package adjudicate

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ttb/labelverify/internal/match"
	"github.com/ttb/labelverify/internal/verify"
)

func TestReviewFieldsShadowsEligibleEquivalentDecisionByDefault(t *testing.T) {
	client := &fakeClient{decision: Decision{
		Decision:    "equivalent",
		Confidence:  0.88,
		MatchedText: "POM CHERRY CORDIAL",
		Reason:      "Cordial is equivalent to the expected class.",
	}}
	service := NewService(testPolicy(), client)
	fields := []match.FieldResult{{
		Field: "class_type", Expected: "Cherry Liqueur/Cordial", Extracted: "POM CHERRY CORDIAL",
		MatchType: "fuzzy", Score: 0.67, Pass: false, Diff: "mismatch",
	}}

	out := service.ReviewFields(context.Background(), fields, []match.TextRegion{{Text: "POM CHERRY CORDIAL", Confidence: 0.99}})

	if out[0].Pass {
		t.Fatalf("shadow adjudication must not change pass/fail: %+v", out[0])
	}
	if out[0].ReviewSource != "llm_shadow" || out[0].ReviewDecision != "equivalent" || out[0].DeterministicScore != 0.67 {
		t.Fatalf("review metadata not set: %+v", out[0])
	}
	if !out[0].ReviewAccepted {
		t.Fatalf("equivalent high-confidence decision should be marked accepted: %+v", out[0])
	}
	if client.calls != 1 {
		t.Fatalf("calls=%d want 1", client.calls)
	}
}

func TestReviewFieldsCanOverrideWhenPolicyAllows(t *testing.T) {
	client := &fakeClient{decision: Decision{
		Decision:    "equivalent",
		Confidence:  0.88,
		MatchedText: "POM CHERRY CORDIAL",
		Reason:      "Cordial is equivalent to the expected class.",
	}}
	policy := testPolicy()
	policy.Mode = "override"
	service := NewService(policy, client)
	fields := []match.FieldResult{{
		Field: "class_type", Expected: "Cherry Liqueur/Cordial", Extracted: "POM CHERRY CORDIAL",
		MatchType: "fuzzy", Score: 0.67, Pass: false, Diff: "mismatch",
	}}

	out := service.ReviewFields(context.Background(), fields, []match.TextRegion{{Text: "POM CHERRY CORDIAL", Confidence: 0.99}})

	if !out[0].Pass || out[0].ReviewSource != "llm" {
		t.Fatalf("override adjudication should pass with llm source: %+v", out[0])
	}
}

func TestReviewFieldsDoesNotCallDeniedOrOutOfRangeFields(t *testing.T) {
	client := &fakeClient{decision: Decision{Decision: "equivalent", Confidence: 1}}
	service := NewService(testPolicy(), client)
	fields := []match.FieldResult{
		{Field: "abv", MatchType: "format", Score: 0.7, Pass: false},
		{Field: "brand", MatchType: "fuzzy", Score: 0.59, Pass: false},
		{Field: "class_type", MatchType: "fuzzy", Score: 0.96, Pass: false},
		{Field: "government_warning", MatchType: "exact", Score: 0.8, Pass: false},
	}

	out := service.ReviewFields(context.Background(), fields, nil)

	if client.calls != 0 {
		t.Fatalf("calls=%d want 0", client.calls)
	}
	for _, field := range out {
		if field.Pass {
			t.Fatalf("field unexpectedly passed: %+v", field)
		}
	}
}

func TestReviewFieldsCanReviewExplicitlyAllowedNonFuzzyField(t *testing.T) {
	client := &fakeClient{decision: Decision{Decision: "equivalent", Confidence: 1}}
	policy := testPolicy()
	policy.AllowedFields = map[string]bool{"government_warning": true}
	policy.DeniedFields = map[string]bool{}
	policy.MinDeterministicScore = 0
	policy.MaxDeterministicScore = 1
	service := NewService(policy, client)
	fields := []match.FieldResult{
		{Field: "government_warning", MatchType: "warning", Score: 0, Pass: false},
	}

	out := service.ReviewFields(context.Background(), fields, nil)

	if client.calls != 1 {
		t.Fatalf("calls=%d want 1", client.calls)
	}
	if out[0].ReviewSource != "llm_shadow" || out[0].ReviewDecision != "equivalent" {
		t.Fatalf("field was not reviewed: %+v", out[0])
	}
}

func TestReviewFieldsDoesNotCallBelowEligibleFailingFieldGate(t *testing.T) {
	client := &fakeClient{decision: Decision{Decision: "equivalent", Confidence: 1}}
	policy := testPolicy()
	policy.MinEligibleFailingFields = 2
	service := NewService(policy, client)
	fields := []match.FieldResult{
		{Field: "brand", MatchType: "fuzzy", Score: 0.7, Pass: false},
		{Field: "net_contents", MatchType: "format", Score: 0, Pass: false},
		{Field: "abv", MatchType: "format", Score: 0, Pass: false},
	}

	out := service.ReviewFields(context.Background(), fields, nil)

	if client.calls != 0 {
		t.Fatalf("calls=%d want 0", client.calls)
	}
	if out[0].ReviewSource != "" {
		t.Fatalf("field unexpectedly reviewed: %+v", out[0])
	}
}

func TestReviewFieldsCallsWhenOneEligibleFieldFailsByDefault(t *testing.T) {
	client := &fakeClient{decision: Decision{Decision: "equivalent", Confidence: 1}}
	service := NewService(testPolicy(), client)
	fields := []match.FieldResult{
		{Field: "brand", MatchType: "fuzzy", Score: 0.7, Pass: false},
		{Field: "net_contents", MatchType: "format", Score: 0, Pass: false},
	}

	_ = service.ReviewFields(context.Background(), fields, nil)

	if client.calls != 1 {
		t.Fatalf("calls=%d want 1", client.calls)
	}
}

func TestReviewFieldsFallsBackOnUncertainOrError(t *testing.T) {
	client := &fakeClient{decision: Decision{Decision: "uncertain", Confidence: 0.99}}
	service := NewService(testPolicy(), client)
	field := match.FieldResult{Field: "brand", MatchType: "fuzzy", Score: 0.7, Pass: false, Diff: "mismatch"}

	out := service.ReviewFields(context.Background(), []match.FieldResult{field}, nil)
	if out[0].Pass {
		t.Fatalf("uncertain decision must not pass: %+v", out[0])
	}
	if out[0].ReviewDecision != "uncertain" || out[0].ReviewAccepted {
		t.Fatalf("uncertain decision should be recorded but not accepted: %+v", out[0])
	}

	client = &fakeClient{err: errors.New("timeout")}
	service = NewService(testPolicy(), client)
	out = service.ReviewFields(context.Background(), []match.FieldResult{field}, nil)
	if out[0].Pass {
		t.Fatalf("client error must not pass: %+v", out[0])
	}
	if out[0].ReviewSource != "llm_error" || out[0].ReviewDecision != "error" || !strings.Contains(out[0].ReviewReason, "timeout") {
		t.Fatalf("client error should leave visible error metadata: %+v", out[0])
	}
}

func TestReviewFieldsRecordsNotEquivalentDecision(t *testing.T) {
	client := &fakeClient{decision: Decision{Decision: "not_equivalent", Confidence: 0.93, Reason: "different producer"}}
	service := NewService(testPolicy(), client)
	field := match.FieldResult{Field: "brand", MatchType: "fuzzy", Score: 0.7, Pass: false, Diff: "mismatch"}

	out := service.ReviewFields(context.Background(), []match.FieldResult{field}, nil)
	if out[0].Pass || out[0].ReviewAccepted {
		t.Fatalf("not_equivalent decision must not pass or be accepted: %+v", out[0])
	}
	if out[0].ReviewDecision != "not_equivalent" || out[0].ReviewSource != "llm_shadow" {
		t.Fatalf("not_equivalent metadata not recorded: %+v", out[0])
	}
}

func TestImageReviewerCanOverrideFailedFieldFromParallelRead(t *testing.T) {
	policy := testPolicy()
	policy.Mode = "override"
	client := &fakeLabelClient{read: LabelRead{ClassType: "Bourbon"}}
	reviewer := NewImageReviewer(policy, client)
	future := reviewer.StartReview(context.Background(), verify.Application{ClassType: "Bourbon"}, []byte("image"))
	fields := []match.FieldResult{{
		Field: "class_type", Expected: "Bourbon", Extracted: "DISTILLED FROM MASH",
		MatchType: "fuzzy", Score: 0.2, Pass: false, Diff: "mismatch",
	}}

	out := future.ReviewFields(context.Background(), fields)

	if !out[0].Pass || out[0].Extracted != "Bourbon" || out[0].ReviewExtracted != "Bourbon" || out[0].ReviewSource != "ai_reader" {
		t.Fatalf("AI reader should override with matching extracted text: %+v", out[0])
	}
	if client.calls != 1 {
		t.Fatalf("calls=%d want 1", client.calls)
	}
}

func TestImageReviewerShadowRecordsWithoutChangingVerdict(t *testing.T) {
	client := &fakeLabelClient{read: LabelRead{ClassType: "Bourbon"}}
	reviewer := NewImageReviewer(testPolicy(), client)
	future := reviewer.StartReview(context.Background(), verify.Application{ClassType: "Bourbon"}, []byte("image"))
	fields := []match.FieldResult{{
		Field: "class_type", Expected: "Bourbon", Extracted: "DISTILLED FROM MASH",
		MatchType: "fuzzy", Score: 0.2, Pass: false, Diff: "mismatch",
	}}

	out := future.ReviewFields(context.Background(), fields)

	if out[0].Pass || out[0].ReviewSource != "ai_reader_shadow" || out[0].ReviewDecision != "equivalent" || out[0].ReviewExtracted != "Bourbon" {
		t.Fatalf("AI reader shadow should record but not pass: %+v", out[0])
	}
}

func TestImageReviewerRecordsUncertainWhenFieldNotRead(t *testing.T) {
	policy := testPolicy()
	policy.AllowedFields = map[string]bool{"net_contents": true}
	policy.DeniedFields = map[string]bool{}
	client := &fakeLabelClient{read: LabelRead{}}
	reviewer := NewImageReviewer(policy, client)
	future := reviewer.StartReview(context.Background(), verify.Application{NetContents: "750 mL"}, []byte("image"))
	fields := []match.FieldResult{{
		Field: "net_contents", Expected: "750 mL", Extracted: "",
		MatchType: "format", Score: 0, Pass: false, Diff: "missing",
	}}

	out := future.ReviewFields(context.Background(), fields)

	if out[0].ReviewSource != "ai_reader_shadow" || out[0].ReviewDecision != "uncertain" || out[0].Pass {
		t.Fatalf("missing AI field should be recorded as uncertain: %+v", out[0])
	}
}

func testPolicy() Policy {
	return Policy{
		Enabled: true,
		Mode:    "shadow",
		AllowedFields: map[string]bool{
			"brand":      true,
			"class_type": true,
		},
		DeniedFields: map[string]bool{
			"abv":                true,
			"net_contents":       true,
			"government_warning": true,
			"name_address":       true,
		},
		MinDeterministicScore:    0.60,
		MaxDeterministicScore:    0.95,
		MinLLMConfidence:         0.75,
		MinEligibleFailingFields: 1,
	}
}

type fakeClient struct {
	decision Decision
	err      error
	calls    int
}

func (f *fakeClient) Adjudicate(ctx context.Context, req Request) (Decision, error) {
	f.calls++
	return f.decision, f.err
}

type fakeLabelClient struct {
	read  LabelRead
	err   error
	calls int
}

func (f *fakeLabelClient) ReadLabel(ctx context.Context, app verify.Application, image []byte) (LabelRead, error) {
	f.calls++
	return f.read, f.err
}
