package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ttb/labelverify/internal/adjudicate"
	"github.com/ttb/labelverify/internal/eval"
)

func main() {
	casesDir := flag.String("cases", "../../evals/cases", "directory containing eval case JSON files")
	jsonReport := flag.String("json-report", "../../evals/reports/latest.json", "path to write JSON report; empty disables")
	mdReport := flag.String("md-report", "../../evals/reports/latest.md", "path to write Markdown report; empty disables")
	judgeURL := flag.String("judge-url", "", "optional judge service URL for shadow adjudication during cached eval")
	judgeTimeoutMS := flag.Int("judge-timeout-ms", 500, "judge service timeout in milliseconds")
	flag.Parse()

	opts := eval.Options{CasesDir: *casesDir}
	if *judgeURL != "" {
		if *judgeTimeoutMS <= 0 {
			log.Fatal("judge-timeout-ms must be > 0")
		}
		opts.Adjudicator = adjudicate.NewService(evalJudgePolicy(), adjudicate.NewHTTPClient(*judgeURL, time.Duration(*judgeTimeoutMS)*time.Millisecond))
	}
	report, err := eval.Run(context.Background(), opts)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll("../../evals/reports", 0o700); err != nil {
		log.Fatal(err)
	}
	if err := eval.WriteJSONReport(*jsonReport, report); err != nil {
		log.Fatal(err)
	}
	if err := eval.WriteMarkdownReport(*mdReport, report); err != nil {
		log.Fatal(err)
	}
	fmt.Println(eval.ConsoleSummary(report))
	if report.Summary.FailedCases > 0 || report.Summary.FalsePasses > 0 {
		os.Exit(1)
	}
}

func evalJudgePolicy() adjudicate.Policy {
	return adjudicate.Policy{
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
