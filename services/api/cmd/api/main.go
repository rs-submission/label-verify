package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/ttb/labelverify/internal/adjudicate"
	"github.com/ttb/labelverify/internal/config"
	apihttp "github.com/ttb/labelverify/internal/http"
	"github.com/ttb/labelverify/internal/ocr"
	"github.com/ttb/labelverify/internal/storage"
	"github.com/ttb/labelverify/internal/store"
	"github.com/ttb/labelverify/internal/verify"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := store.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations(ctx, cfg.MigrationsDir); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	ocrClient := ocr.NewClient(cfg.OCRAddr, 5*time.Second)
	imageStore := storage.NewFileStore(cfg.StorageDir)
	verifyTimeout := time.Duration(cfg.VerifyTimeoutMS) * time.Millisecond
	verifier := verify.NewService(db, ocrClient).WithImageStore(imageStore).WithTimeout(verifyTimeout)
	if cfg.LLM.Enabled {
		log.Printf("llm adjudication default enabled mode=%s model=%s addr=%s timeout=%dms", cfg.LLM.Mode, cfg.LLM.Model, cfg.LLM.Addr, cfg.LLM.TimeoutMS)
	}
	router := apihttp.NewRouterWithOptions(db, verifier, imageStore, apihttp.RouterOptions{
		Judge: apihttp.JudgeRuntime{
			DefaultEnabled: cfg.LLM.Enabled,
			Addr:           cfg.LLM.Addr,
			TimeoutMS:      cfg.LLM.TimeoutMS,
			Policy:         adjudicate.PolicyFromConfig(cfg.LLM),
		},
	})

	log.Printf("api listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal(err)
	}
}
