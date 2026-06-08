package adjudicate

import (
	"context"
	"log"
	"strings"

	"github.com/ttb/labelverify/internal/config"
	"github.com/ttb/labelverify/internal/match"
)

type Request struct {
	Field              string   `json:"field"`
	Expected           string   `json:"expected"`
	Extracted          string   `json:"extracted"`
	DeterministicScore float64  `json:"deterministic_score"`
	OCRCandidates      []string `json:"ocr_candidates"`
}

type Decision struct {
	Decision    string  `json:"decision"`
	Confidence  float64 `json:"confidence"`
	MatchedText string  `json:"matched_text"`
	Reason      string  `json:"reason"`
}

type Client interface {
	Adjudicate(ctx context.Context, req Request) (Decision, error)
}

type Policy struct {
	Enabled                  bool
	Mode                     string
	AllowedFields            map[string]bool
	DeniedFields             map[string]bool
	MinDeterministicScore    float64
	MaxDeterministicScore    float64
	MinLLMConfidence         float64
	MinEligibleFailingFields int
}

type Service struct {
	policy Policy
	client Client
}

func PolicyFromConfig(cfg config.LLMConfig) Policy {
	return Policy{
		Enabled:                  cfg.Enabled,
		Mode:                     cfg.Mode,
		AllowedFields:            cfg.AllowedFields,
		DeniedFields:             cfg.DeniedFields,
		MinDeterministicScore:    0,
		MaxDeterministicScore:    1,
		MinLLMConfidence:         0.75,
		MinEligibleFailingFields: cfg.MinEligibleFailingFields,
	}
}

func NewService(policy Policy, client Client) *Service {
	return &Service{policy: policy, client: client}
}

func (s *Service) ReviewFields(ctx context.Context, fields []match.FieldResult, pool []match.TextRegion) []match.FieldResult {
	if s == nil || s.client == nil || !s.policy.Enabled {
		return fields
	}
	if eligibleFailingFieldCount(fields, s.eligible) < s.policy.MinEligibleFailingFields {
		return fields
	}

	out := append([]match.FieldResult(nil), fields...)
	candidates := candidateTexts(pool)
	for i, field := range out {
		if !s.eligible(field) {
			continue
		}
		decision, err := s.client.Adjudicate(ctx, Request{
			Field:              field.Field,
			Expected:           field.Expected,
			Extracted:          field.Extracted,
			DeterministicScore: field.Score,
			OCRCandidates:      candidates,
		})
		if err != nil {
			log.Printf("judge review failed for field %q: %v", field.Field, err)
			out[i].DeterministicScore = field.Score
			out[i].ReviewSource = "llm_error"
			out[i].ReviewDecision = "error"
			out[i].ReviewReason = err.Error()
			continue
		}

		accepted := s.accepts(decision)
		out[i].DeterministicScore = field.Score
		out[i].ReviewSource = "llm_shadow"
		out[i].ReviewDecision = normalizedDecision(decision.Decision)
		out[i].ReviewConfidence = decision.Confidence
		out[i].ReviewReason = strings.TrimSpace(decision.Reason)
		out[i].ReviewAccepted = accepted
		if !accepted {
			continue
		}
		if s.policy.Mode == "override" {
			out[i].Score = max(out[i].Score, decision.Confidence)
			out[i].Pass = true
			out[i].Diff = ""
			out[i].ReviewSource = "llm"
			if strings.TrimSpace(decision.MatchedText) != "" {
				out[i].Extracted = strings.TrimSpace(decision.MatchedText)
			}
		}
	}
	return out
}

func normalizedDecision(value string) string {
	decision := strings.ToLower(strings.TrimSpace(value))
	switch decision {
	case "equivalent", "not_equivalent", "uncertain":
		return decision
	default:
		return "uncertain"
	}
}

func eligibleFailingFieldCount(fields []match.FieldResult, eligible func(match.FieldResult) bool) int {
	count := 0
	for _, field := range fields {
		if eligible(field) {
			count++
		}
	}
	return count
}

func (s *Service) eligible(field match.FieldResult) bool {
	if field.Pass {
		return false
	}
	if s.policy.DeniedFields[field.Field] {
		return false
	}
	if !s.policy.AllowedFields[field.Field] {
		return false
	}
	return field.Score >= s.policy.MinDeterministicScore && field.Score <= s.policy.MaxDeterministicScore
}

func (s *Service) accepts(decision Decision) bool {
	return strings.EqualFold(strings.TrimSpace(decision.Decision), "equivalent") &&
		decision.Confidence >= s.policy.MinLLMConfidence
}

func candidateTexts(pool []match.TextRegion) []string {
	candidates := match.AssembleCandidateWindows(pool, 0.25)
	seen := make(map[string]bool, len(candidates))
	out := make([]string, 0, minInt(len(candidates), 20))
	for _, candidate := range candidates {
		text := strings.TrimSpace(candidate.Text)
		key := match.NormalizeFlexible(text)
		if text == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, text)
		if len(out) >= 20 {
			break
		}
	}
	return out
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
