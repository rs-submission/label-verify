package ocr

import (
	"encoding/json"
	"os"
	"testing"
)

func TestGoldenResponseContract(t *testing.T) {
	data, err := os.ReadFile("../../../../contracts/ocr_response.golden.json")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var response RecognizeResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}
	if response.ElapsedMS == 0 {
		t.Fatal("elapsed_ms must be populated")
	}
	if len(response.Regions) != 1 {
		t.Fatalf("regions length=%d want 1", len(response.Regions))
	}
	if response.Regions[0].Text == "" || response.Regions[0].Confidence == 0 {
		t.Fatalf("region not populated: %+v", response.Regions[0])
	}
}
