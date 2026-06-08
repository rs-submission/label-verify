DOCKER_CONFIG_DIR := $(CURDIR)/.cache/docker-config

.PHONY: dev test eval

dev:
	mkdir -p "$(DOCKER_CONFIG_DIR)"
	test -f "$(DOCKER_CONFIG_DIR)/config.json" || printf '{"auths":{}}\n' > "$(DOCKER_CONFIG_DIR)/config.json"
	DOCKER_CONFIG="$(DOCKER_CONFIG_DIR)" docker-compose up --build

test:
	cd services/api && GOCACHE=$$(pwd)/../../.cache/go-build go test ./...
	cd services/ocr && UV_CACHE_DIR=$$(pwd)/../../.cache/uv uv run pytest

eval:
	mkdir -p evals/reports
	cd services/api && GOCACHE=$$(pwd)/../../.cache/go-build go run ./cmd/eval --cases ../../evals/cases --json-report ../../evals/reports/latest.json --md-report ../../evals/reports/latest.md
