package config

import "testing"

func TestLoadDefaultsLLMPolicy(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("LLM_ALLOWED_FIELDS", "")
	t.Setenv("LLM_DENIED_FIELDS", "")
	t.Setenv("LLM_MIN_ELIGIBLE_FAILING_FIELDS", "")
	t.Setenv("LLM_TIMEOUT_MS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.LLM.Enabled {
		t.Fatal("LLM must default disabled")
	}
	if cfg.LLM.Model != "gemma4:latest" {
		t.Fatalf("model=%q want gemma4:latest", cfg.LLM.Model)
	}
	if cfg.LLM.Mode != "shadow" {
		t.Fatalf("mode=%q want shadow", cfg.LLM.Mode)
	}
	if !cfg.LLM.AllowedFields["brand"] || !cfg.LLM.AllowedFields["class_type"] {
		t.Fatalf("allowed fields=%v want brand,class_type", cfg.LLM.AllowedFields)
	}
	if !cfg.LLM.DeniedFields["abv"] || !cfg.LLM.DeniedFields["net_contents"] || !cfg.LLM.DeniedFields["government_warning"] || !cfg.LLM.DeniedFields["name_address"] {
		t.Fatalf("denied fields=%v want deterministic-only fields denied", cfg.LLM.DeniedFields)
	}
	if cfg.LLM.Addr != "http://localhost:8002" {
		t.Fatalf("addr=%q want local judge service", cfg.LLM.Addr)
	}
	if cfg.LLM.MinEligibleFailingFields != 1 {
		t.Fatalf("min eligible failing fields=%d want 1", cfg.LLM.MinEligibleFailingFields)
	}
	if cfg.LLM.TimeoutMS != 4500 {
		t.Fatalf("timeout ms=%d want 4500", cfg.LLM.TimeoutMS)
	}
	if cfg.StorageDir != "./data/images" {
		t.Fatalf("StorageDir=%q want ./data/images", cfg.StorageDir)
	}
	if cfg.VerifyTimeoutMS != 5000 {
		t.Fatalf("VerifyTimeoutMS=%d want 5000", cfg.VerifyTimeoutMS)
	}
}

func TestLoadOverridesLLMPolicy(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("LLM_ENABLED", "true")
	t.Setenv("LLM_MODE", "override")
	t.Setenv("LLM_ADDR", "http://localhost:11434")
	t.Setenv("LLM_MODEL", "llama3.2")
	t.Setenv("LLM_ALLOWED_FIELDS", "brand,producer")
	t.Setenv("LLM_DENIED_FIELDS", "abv")
	t.Setenv("LLM_MIN_ELIGIBLE_FAILING_FIELDS", "2")
	t.Setenv("LLM_TIMEOUT_MS", "750")
	t.Setenv("VERIFY_TIMEOUT_MS", "4500")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if !cfg.LLM.Enabled || cfg.LLM.Mode != "override" || cfg.LLM.Addr != "http://localhost:11434" || cfg.LLM.Model != "llama3.2" {
		t.Fatalf("llm config=%+v", cfg.LLM)
	}
	if !cfg.LLM.AllowedFields["brand"] || !cfg.LLM.AllowedFields["producer"] || cfg.LLM.AllowedFields["class_type"] {
		t.Fatalf("allowed fields=%v", cfg.LLM.AllowedFields)
	}
	if !cfg.LLM.DeniedFields["abv"] || cfg.LLM.DeniedFields["government_warning"] {
		t.Fatalf("denied fields=%v", cfg.LLM.DeniedFields)
	}
	if cfg.LLM.MinEligibleFailingFields != 2 {
		t.Fatalf("min eligible failing fields=%d", cfg.LLM.MinEligibleFailingFields)
	}
	if cfg.LLM.TimeoutMS != 750 {
		t.Fatalf("timeout ms=%d", cfg.LLM.TimeoutMS)
	}
	if cfg.VerifyTimeoutMS != 4500 {
		t.Fatalf("verify timeout ms=%d", cfg.VerifyTimeoutMS)
	}
}

func TestLoadRejectsInvalidLLMMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("LLM_MODE", "decide-for-me")

	if _, err := Load(); err == nil {
		t.Fatal("Load must reject invalid LLM mode")
	}
}

func TestLoadRejectsNegativeMinEligibleFailingFields(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("LLM_MIN_ELIGIBLE_FAILING_FIELDS", "-1")

	if _, err := Load(); err == nil {
		t.Fatal("Load must reject negative min eligible failing fields")
	}
}

func TestLoadRejectsInvalidLLMTimeout(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("LLM_TIMEOUT_MS", "0")

	if _, err := Load(); err == nil {
		t.Fatal("Load must reject invalid LLM timeout")
	}
}

func TestLoadRejectsInvalidVerifyTimeout(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("VERIFY_TIMEOUT_MS", "0")

	if _, err := Load(); err == nil {
		t.Fatal("Load must reject invalid verify timeout")
	}
}
