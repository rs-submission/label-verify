package adjudicate

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ttb/labelverify/internal/match"
	"github.com/ttb/labelverify/internal/verify"
)

type LabelReadClient interface {
	ReadLabel(ctx context.Context, app verify.Application, image []byte) (LabelRead, error)
}

type ImageReviewer struct {
	policy Policy
	client LabelReadClient
}

func NewImageReviewer(policy Policy, client LabelReadClient) *ImageReviewer {
	return &ImageReviewer{policy: policy, client: client}
}

type imageReviewFuture struct {
	policy Policy
	ch     <-chan labelReadResult
}

type labelReadResult struct {
	read LabelRead
	err  error
}

func (r *ImageReviewer) StartReview(ctx context.Context, app verify.Application, image []byte) verify.ImageReviewFuture {
	ch := make(chan labelReadResult, 1)
	go func() {
		read, err := r.client.ReadLabel(ctx, app, image)
		ch <- labelReadResult{read: read, err: err}
	}()
	return &imageReviewFuture{policy: r.policy, ch: ch}
}

func (f *imageReviewFuture) ReviewFields(ctx context.Context, fields []match.FieldResult) []match.FieldResult {
	if eligibleFailingFieldCount(fields, f.eligible) < f.policy.MinEligibleFailingFields {
		return fields
	}

	out := append([]match.FieldResult(nil), fields...)
	var result labelReadResult
	select {
	case result = <-f.ch:
	case <-ctx.Done():
		result.err = ctx.Err()
	}
	if result.err != nil {
		log.Printf("AI label reader failed: %v", result.err)
		for i, field := range out {
			if f.eligible(field) {
				out[i].DeterministicScore = field.Score
				out[i].ReviewSource = "ai_reader_error"
				out[i].ReviewDecision = "error"
				out[i].ReviewReason = result.err.Error()
			}
		}
		return out
	}

	for i, field := range out {
		if !f.eligible(field) {
			continue
		}
		value := strings.TrimSpace(result.read.valueFor(field.Field))
		out[i].DeterministicScore = field.Score
		out[i].ReviewSource = "ai_reader_shadow"
		out[i].ReviewDecision = "uncertain"
		out[i].ReviewReason = "AI label reader did not find this field."
		out[i].ReviewAccepted = false
		if value == "" {
			continue
		}
		out[i].ReviewExtracted = value
		recheck := evaluateReadValue(field, value)
		out[i].ReviewDecision = "not_equivalent"
		out[i].ReviewReason = fmt.Sprintf("AI label reader extracted %q.", value)
		if !recheck.Pass {
			continue
		}
		out[i].ReviewDecision = "equivalent"
		out[i].ReviewConfidence = 1
		out[i].ReviewAccepted = true
		out[i].ReviewReason = "AI label reader extracted matching label text."
		if f.policy.Mode == "override" {
			out[i].Extracted = value
			out[i].Score = max(out[i].Score, recheck.Score)
			out[i].Pass = true
			out[i].Diff = ""
			out[i].ReviewSource = "ai_reader"
		}
	}
	return out
}

func (f *imageReviewFuture) eligible(field match.FieldResult) bool {
	if field.Pass {
		return false
	}
	if f.policy.DeniedFields[field.Field] {
		return false
	}
	if !f.policy.AllowedFields[field.Field] {
		return false
	}
	return true
}

func (r LabelRead) valueFor(field string) string {
	switch field {
	case "brand":
		return r.Brand
	case "class_type":
		return r.ClassType
	case "net_contents":
		return r.NetContents
	case "abv":
		return r.AlcoholContents
	case "government_warning":
		return r.GovernmentWarning
	case "name_address":
		return r.NameAndAddress
	default:
		return ""
	}
}

func evaluateReadValue(field match.FieldResult, value string) match.FieldResult {
	spec := readValueSpec(field)
	result := match.MatchAssigned([]match.FieldSpec{spec}, []match.TextRegion{{Text: value, Confidence: 1}})
	if len(result) == 0 {
		return match.FieldResult{Field: field.Field, Expected: field.Expected, Extracted: value, Score: 0}
	}
	return result[0]
}

func readValueSpec(field match.FieldResult) match.FieldSpec {
	switch field.Field {
	case "brand":
		return match.FuzzyField(field.Field, field.Expected, 0.85)
	case "class_type":
		return match.FuzzyField(field.Field, field.Expected, 0.95)
	case "name_address":
		return match.NameAddressField(field.Field, field.Expected, 0.85)
	case "government_warning":
		return match.WarningField(field.Field, field.Expected)
	default:
		if field.MatchType == "format" {
			return match.FormatField(field.Field, field.Expected)
		}
		if field.MatchType == "warning" {
			return match.WarningField(field.Field, field.Expected)
		}
		return match.FuzzyField(field.Field, field.Expected, 0.85)
	}
}
