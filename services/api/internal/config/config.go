package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL     string
	OCRAddr         string
	StorageDir      string
	MigrationsDir   string
	Port            string
	VerifyTimeoutMS int
	LLM             LLMConfig
}

type LLMConfig struct {
	Enabled                  bool
	Mode                     string
	Addr                     string
	Model                    string
	TimeoutMS                int
	AllowedFields            map[string]bool
	DeniedFields             map[string]bool
	MinEligibleFailingFields int
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		OCRAddr:         env("OCR_ADDR", "http://localhost:8001"),
		StorageDir:      env("STORAGE_DIR", "./data/images"),
		MigrationsDir:   env("MIGRATIONS_DIR", "migrations"),
		Port:            env("PORT", "8080"),
		VerifyTimeoutMS: envInt("VERIFY_TIMEOUT_MS", 5000),
		LLM: LLMConfig{
			Enabled:                  envBool("LLM_ENABLED", false),
			Mode:                     env("LLM_MODE", "shadow"),
			Addr:                     env("LLM_ADDR", "http://localhost:8002"),
			Model:                    env("LLM_MODEL", "gemma4:latest"),
			TimeoutMS:                envInt("LLM_TIMEOUT_MS", 4500),
			AllowedFields:            envSet("LLM_ALLOWED_FIELDS", "brand,class_type"),
			DeniedFields:             envSet("LLM_DENIED_FIELDS", "abv,net_contents,government_warning,name_address"),
			MinEligibleFailingFields: envInt("LLM_MIN_ELIGIBLE_FAILING_FIELDS", 1),
		},
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.LLM.Mode != "shadow" && cfg.LLM.Mode != "override" {
		return Config{}, fmt.Errorf("LLM_MODE must be shadow or override")
	}
	if cfg.LLM.MinEligibleFailingFields < 0 {
		return Config{}, fmt.Errorf("LLM_MIN_ELIGIBLE_FAILING_FIELDS must be >= 0")
	}
	if cfg.LLM.TimeoutMS <= 0 {
		return Config{}, fmt.Errorf("LLM_TIMEOUT_MS must be > 0")
	}
	if cfg.VerifyTimeoutMS <= 0 {
		return Config{}, fmt.Errorf("VERIFY_TIMEOUT_MS must be > 0")
	}
	return cfg, nil
}

func env(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envSet(name, fallback string) map[string]bool {
	value := env(name, fallback)
	out := make(map[string]bool)
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out[part] = true
		}
	}
	return out
}
