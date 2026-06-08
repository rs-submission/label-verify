package eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ttb/labelverify/internal/match"
	"github.com/ttb/labelverify/internal/ocr"
	"github.com/ttb/labelverify/internal/verify"
)

type Case struct {
	ID          string             `json:"id"`
	Description string             `json:"description"`
	Application verify.Application `json:"application"`
	OCR         OCRFixture         `json:"ocr"`
	Expected    Expected           `json:"expected"`
}

type OCRFixture struct {
	Regions []ocr.Region `json:"regions"`
}

type Expected struct {
	Status string            `json:"status"`
	Fields map[string]string `json:"fields"`
}

type Options struct {
	CasesDir    string
	Adjudicator verify.FieldAdjudicator
}

type Report struct {
	StartedAt      time.Time      `json:"started_at"`
	Mode           string         `json:"mode"`
	LatencyNote    string         `json:"latency_note"`
	DurationMS     int64          `json:"duration_ms"`
	Cases          []CaseResult   `json:"cases"`
	Summary        Summary        `json:"summary"`
	FieldSummaries []FieldSummary `json:"field_summaries"`
	JudgeSummary   JudgeSummary   `json:"judge_summary"`
	LatencyMS      LatencySummary `json:"latency_ms"`
}

type CaseResult struct {
	ID         string             `json:"id"`
	Status     string             `json:"status"`
	Expected   string             `json:"expected"`
	Passed     bool               `json:"passed"`
	DurationMS int64              `json:"duration_ms"`
	Fields     []FieldCheckResult `json:"fields"`
	Error      string             `json:"error,omitempty"`
	Verdict    *match.Verdict     `json:"verdict,omitempty"`
}

type FieldCheckResult struct {
	Field    string             `json:"field"`
	Expected string             `json:"expected"`
	Actual   string             `json:"actual"`
	Passed   bool               `json:"passed"`
	Result   *match.FieldResult `json:"result,omitempty"`
}

type Summary struct {
	CaseCount    int `json:"case_count"`
	PassedCases  int `json:"passed_cases"`
	FailedCases  int `json:"failed_cases"`
	FieldChecks  int `json:"field_checks"`
	PassedFields int `json:"passed_fields"`
	FailedFields int `json:"failed_fields"`
	FalsePasses  int `json:"false_passes"`
	FalseFlags   int `json:"false_flags"`
}

type FieldSummary struct {
	Field       string `json:"field"`
	Checks      int    `json:"checks"`
	Passed      int    `json:"passed"`
	Failed      int    `json:"failed"`
	FalsePasses int    `json:"false_passes"`
	FalseFlags  int    `json:"false_flags"`
}

type LatencySummary struct {
	P50 int64 `json:"p50"`
	P95 int64 `json:"p95"`
	P99 int64 `json:"p99"`
}

type JudgeSummary struct {
	Reviews           int            `json:"reviews"`
	Equivalent        int            `json:"equivalent"`
	NotEquivalent     int            `json:"not_equivalent"`
	Uncertain         int            `json:"uncertain"`
	Accepted          int            `json:"accepted"`
	WouldChangeFields int            `json:"would_change_fields"`
	ByField           map[string]int `json:"by_field,omitempty"`
}

func Run(ctx context.Context, opts Options) (Report, error) {
	started := time.Now()
	cases, err := LoadCases(opts.CasesDir)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		StartedAt:   started,
		Mode:        "cached_ocr",
		LatencyNote: "Cached OCR mode measures verifier/matcher runtime only; it does not include OCR or preprocessing latency.",
	}
	for _, tc := range cases {
		report.Cases = append(report.Cases, runCase(ctx, tc, opts.Adjudicator))
	}
	report.DurationMS = time.Since(started).Milliseconds()
	report.Summary, report.FieldSummaries, report.LatencyMS, report.JudgeSummary = summarize(report.Cases)
	return report, nil
}

func LoadCases(dir string) ([]Case, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("cases dir is required")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)

	cases := make([]Case, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		var tc Case
		if err := json.Unmarshal(data, &tc); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		if err := validateCase(tc); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		cases = append(cases, tc)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("no eval case JSON files found in %s", dir)
	}
	return cases, nil
}

func runCase(ctx context.Context, tc Case, adjudicator verify.FieldAdjudicator) CaseResult {
	started := time.Now()
	store := &caseStore{app: tc.Application}
	ocrClient := &caseOCR{response: ocr.RecognizeResponse{Regions: tc.OCR.Regions}}
	service := verify.NewService(store, ocrClient)
	if adjudicator != nil {
		service = verify.NewServiceWithAdjudicator(store, ocrClient, adjudicator)
	}
	verdict, err := service.VerifySingle(ctx, tc.Application.ID, nil)
	duration := time.Since(started).Milliseconds()
	result := CaseResult{
		ID:         tc.ID,
		Expected:   tc.Expected.Status,
		DurationMS: duration,
	}
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result
	}
	result.Status = verdict.Status
	result.Passed = verdict.Status == tc.Expected.Status
	result.Verdict = &verdict
	result.Fields = compareFields(tc.Expected.Fields, verdict.Fields)
	for _, field := range result.Fields {
		if !field.Passed {
			result.Passed = false
			break
		}
	}
	return result
}

func compareFields(expected map[string]string, actual []match.FieldResult) []FieldCheckResult {
	byField := make(map[string]match.FieldResult, len(actual))
	for _, field := range actual {
		byField[field.Field] = field
	}
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]FieldCheckResult, 0, len(names))
	for _, name := range names {
		actualResult, ok := byField[name]
		actualState := "missing"
		if ok {
			actualState = passState(actualResult.Pass)
		}
		check := FieldCheckResult{
			Field:    name,
			Expected: expected[name],
			Actual:   actualState,
			Passed:   actualState == expected[name],
		}
		if ok {
			copy := actualResult
			check.Result = &copy
		}
		out = append(out, check)
	}
	return out
}

func summarize(results []CaseResult) (Summary, []FieldSummary, LatencySummary, JudgeSummary) {
	summary := Summary{CaseCount: len(results)}
	fieldStats := make(map[string]FieldSummary)
	latencies := make([]int64, 0, len(results))
	judge := JudgeSummary{ByField: make(map[string]int)}

	for _, result := range results {
		latencies = append(latencies, result.DurationMS)
		if result.Passed {
			summary.PassedCases++
		} else {
			summary.FailedCases++
		}
		for _, field := range result.Fields {
			summary.FieldChecks++
			stat := fieldStats[field.Field]
			stat.Field = field.Field
			stat.Checks++
			if field.Passed {
				summary.PassedFields++
				stat.Passed++
			} else {
				summary.FailedFields++
				stat.Failed++
			}
			if field.Expected == "fail" && field.Actual == "pass" {
				summary.FalsePasses++
				stat.FalsePasses++
			}
			if field.Expected == "pass" && field.Actual == "fail" {
				summary.FalseFlags++
				stat.FalseFlags++
			}
			fieldStats[field.Field] = stat
		}
		if result.Verdict != nil {
			accumulateJudgeSummary(&judge, result.Verdict.Fields)
		}
	}

	fields := make([]FieldSummary, 0, len(fieldStats))
	for _, stat := range fieldStats {
		fields = append(fields, stat)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Field < fields[j].Field })
	if judge.Reviews == 0 {
		judge.ByField = nil
	}
	return summary, fields, latencySummary(latencies), judge
}

func accumulateJudgeSummary(summary *JudgeSummary, fields []match.FieldResult) {
	for _, field := range fields {
		if field.ReviewSource == "" {
			continue
		}
		summary.Reviews++
		summary.ByField[field.Field]++
		switch field.ReviewDecision {
		case "equivalent":
			summary.Equivalent++
		case "not_equivalent":
			summary.NotEquivalent++
		default:
			summary.Uncertain++
		}
		if field.ReviewAccepted {
			summary.Accepted++
			if !field.Pass {
				summary.WouldChangeFields++
			}
		}
	}
}

func latencySummary(values []int64) LatencySummary {
	if len(values) == 0 {
		return LatencySummary{}
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return LatencySummary{
		P50: percentile(values, 0.50),
		P95: percentile(values, 0.95),
		P99: percentile(values, 0.99),
	}
}

func percentile(values []int64, p float64) int64 {
	if len(values) == 0 {
		return 0
	}
	index := int(math.Ceil(p*float64(len(values)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func validateCase(tc Case) error {
	if strings.TrimSpace(tc.ID) == "" {
		return errors.New("id is required")
	}
	if strings.TrimSpace(tc.Application.ID) == "" {
		return errors.New("application.ID is required")
	}
	if strings.TrimSpace(tc.Expected.Status) == "" {
		return errors.New("expected.status is required")
	}
	for field, expected := range tc.Expected.Fields {
		if expected != "pass" && expected != "fail" {
			return fmt.Errorf("expected.fields.%s must be pass or fail", field)
		}
	}
	return nil
}

func passState(pass bool) string {
	if pass {
		return "pass"
	}
	return "fail"
}

type caseStore struct {
	app verify.Application
}

func (s *caseStore) GetApplication(ctx context.Context, id string) (verify.Application, error) {
	if id != s.app.ID {
		return verify.Application{}, fmt.Errorf("unknown application %s", id)
	}
	return s.app, nil
}

func (s *caseStore) SaveVerification(ctx context.Context, result verify.StoredVerification) error {
	return nil
}

type caseOCR struct {
	response ocr.RecognizeResponse
}

func (c *caseOCR) Recognize(ctx context.Context, image []byte, langs []string) (ocr.RecognizeResponse, error) {
	return c.response, nil
}
