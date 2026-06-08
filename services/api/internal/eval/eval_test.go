package eval

import (
	"context"
	"os"
	"testing"

	"github.com/ttb/labelverify/internal/match"
)

func TestRunEvaluatesCachedOCRCase(t *testing.T) {
	dir := t.TempDir()
	writeCase(t, dir, `{
	  "id": "sample",
	  "application": {
	    "ID": "app-1",
	    "Brand": "POM CREEK DISTILLING COMPANY, LLC",
	    "ClassType": "Cherry Liqueur/Cordial",
	    "NetContents": "750 mL",
	    "ABV": "25% ALC/VOL(50 Proof)",
	    "GovernmentWarning": "GOVERNMENT WARNING"
	  },
	  "ocr": {
	    "regions": [
	      {"text":"POM CHERRY CORDIAL","confidence":0.99,"bbox":[7,11,1218,156]},
	      {"text":"POMCREEKDISTILLING COMPANY,LLC","confidence":0.99,"bbox":[379,1294,711,34]},
	      {"text":"25% ALC.BYVOL. (50 PR00F)","confidence":0.90,"bbox":[552,1470,536,38]}
	    ]
	  },
	  "expected": {
	    "status": "flagged",
	    "fields": {
	      "brand": "pass",
	      "class_type": "pass",
	      "net_contents": "fail",
	      "abv": "pass",
	      "government_warning": "fail"
	    }
	  }
	}`)

	report, err := Run(context.Background(), Options{CasesDir: dir})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if report.Summary.CaseCount != 1 || report.Summary.PassedCases != 1 || report.Summary.FalsePasses != 0 {
		t.Fatalf("summary=%+v", report.Summary)
	}
}

func TestRunSummarizesJudgeShadowMetadata(t *testing.T) {
	dir := t.TempDir()
	writeCase(t, dir, `{
	  "id": "sample",
	  "application": {
	    "ID": "app-1",
	    "Brand": "POM CREEK DISTILLING COMPANY, LLC",
	    "ClassType": "Cherry Liqueur/Cordial",
	    "NetContents": "750 mL",
	    "ABV": "25% ALC/VOL(50 Proof)",
	    "GovernmentWarning": "GOVERNMENT WARNING"
	  },
	  "ocr": {
	    "regions": [
	      {"text":"POM CHERRY CORDIAL","confidence":0.99,"bbox":[7,11,1218,156]},
	      {"text":"POMCREEKDISTILLING COMPANY,LLC","confidence":0.99,"bbox":[379,1294,711,34]},
	      {"text":"25% ALC.BYVOL. (50 PR00F)","confidence":0.90,"bbox":[552,1470,536,38]}
	    ]
	  },
	  "expected": {
	    "status": "flagged",
	    "fields": {
	      "brand": "pass",
	      "class_type": "pass",
	      "net_contents": "fail",
	      "abv": "pass",
	      "government_warning": "fail"
	    }
	  }
	}`)

	report, err := Run(context.Background(), Options{CasesDir: dir, Adjudicator: fakeEvalAdjudicator{}})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if report.JudgeSummary.Reviews != 1 || report.JudgeSummary.Equivalent != 1 || report.JudgeSummary.Accepted != 1 {
		t.Fatalf("judge summary=%+v", report.JudgeSummary)
	}
	if report.JudgeSummary.ByField["class_type"] != 1 {
		t.Fatalf("judge by field=%+v", report.JudgeSummary.ByField)
	}
}

func TestCompareFieldsDetectsFalsePass(t *testing.T) {
	checks := compareFields(map[string]string{"abv": "fail"}, []match.FieldResult{matchField("abv", true)})
	if len(checks) != 1 || checks[0].Expected != "fail" || checks[0].Actual != "pass" || checks[0].Passed {
		t.Fatalf("checks=%+v", checks)
	}
}

func writeCase(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(dir+"/sample.json", []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func matchField(field string, pass bool) match.FieldResult {
	return match.FieldResult{Field: field, Pass: pass}
}

type fakeEvalAdjudicator struct{}

func (fakeEvalAdjudicator) ReviewFields(ctx context.Context, fields []match.FieldResult, pool []match.TextRegion) []match.FieldResult {
	out := append([]match.FieldResult(nil), fields...)
	for i := range out {
		if out[i].Field == "class_type" {
			out[i].ReviewSource = "llm_shadow"
			out[i].ReviewDecision = "equivalent"
			out[i].ReviewConfidence = 0.91
			out[i].ReviewAccepted = true
			out[i].DeterministicScore = out[i].Score
			break
		}
	}
	return out
}
