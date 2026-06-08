package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func WriteJSONReport(path string, report Report) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func WriteMarkdownReport(path string, report Report) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Label Verification Eval Report\n\n")
	fmt.Fprintf(&b, "- Mode: %s\n", report.Mode)
	fmt.Fprintf(&b, "- Cases: %d\n", report.Summary.CaseCount)
	fmt.Fprintf(&b, "- Passed cases: %d\n", report.Summary.PassedCases)
	fmt.Fprintf(&b, "- Failed cases: %d\n", report.Summary.FailedCases)
	fmt.Fprintf(&b, "- Field checks: %d\n", report.Summary.FieldChecks)
	fmt.Fprintf(&b, "- False passes: %d\n", report.Summary.FalsePasses)
	fmt.Fprintf(&b, "- False flags: %d\n", report.Summary.FalseFlags)
	fmt.Fprintf(&b, "- Cached matcher latency p50/p95/p99 ms: %d/%d/%d\n", report.LatencyMS.P50, report.LatencyMS.P95, report.LatencyMS.P99)
	fmt.Fprintf(&b, "- Latency note: %s\n\n", report.LatencyNote)
	if report.JudgeSummary.Reviews > 0 {
		fmt.Fprintf(&b, "## Judge Shadow Summary\n\n")
		fmt.Fprintf(&b, "- Reviews: %d\n", report.JudgeSummary.Reviews)
		fmt.Fprintf(&b, "- Equivalent: %d\n", report.JudgeSummary.Equivalent)
		fmt.Fprintf(&b, "- Not equivalent: %d\n", report.JudgeSummary.NotEquivalent)
		fmt.Fprintf(&b, "- Uncertain: %d\n", report.JudgeSummary.Uncertain)
		fmt.Fprintf(&b, "- Accepted: %d\n", report.JudgeSummary.Accepted)
		fmt.Fprintf(&b, "- Would change fields in override mode: %d\n\n", report.JudgeSummary.WouldChangeFields)
	}

	fmt.Fprintf(&b, "## Field Metrics\n\n")
	fmt.Fprintf(&b, "| Field | Checks | Passed | Failed | False Passes | False Flags |\n")
	fmt.Fprintf(&b, "| --- | ---: | ---: | ---: | ---: | ---: |\n")
	for _, field := range report.FieldSummaries {
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %d |\n", field.Field, field.Checks, field.Passed, field.Failed, field.FalsePasses, field.FalseFlags)
	}

	fmt.Fprintf(&b, "\n## Cases\n\n")
	fmt.Fprintf(&b, "| Case | Expected | Actual | Passed | Cached matcher ms |\n")
	fmt.Fprintf(&b, "| --- | --- | --- | --- | ---: |\n")
	for _, tc := range report.Cases {
		fmt.Fprintf(&b, "| %s | %s | %s | %t | %d |\n", tc.ID, tc.Expected, tc.Status, tc.Passed, tc.DurationMS)
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

func ConsoleSummary(report Report) string {
	summary := fmt.Sprintf(
		"mode=%s cases=%d passed=%d failed=%d field_checks=%d false_passes=%d false_flags=%d cached_matcher_latency_ms_p50=%d p95=%d p99=%d",
		report.Mode,
		report.Summary.CaseCount,
		report.Summary.PassedCases,
		report.Summary.FailedCases,
		report.Summary.FieldChecks,
		report.Summary.FalsePasses,
		report.Summary.FalseFlags,
		report.LatencyMS.P50,
		report.LatencyMS.P95,
		report.LatencyMS.P99,
	)
	if report.JudgeSummary.Reviews > 0 {
		summary += fmt.Sprintf(
			" judge_reviews=%d judge_accepted=%d judge_would_change_fields=%d",
			report.JudgeSummary.Reviews,
			report.JudgeSummary.Accepted,
			report.JudgeSummary.WouldChangeFields,
		)
	}
	return summary
}
