.PHONY: build test lint run run-dev release smoke docker-build fmt

BINARY := agentary
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build binary with embedded React UI (requires Node for build-web).
build: build-web
	go mod tidy
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o bin/$(BINARY) ./cmd/agentary

build-web:
	cd web && npm ci && npm run build
	rm -rf internal/ui/dist && cp -r web/dist internal/ui/dist

test:
	go test ./cmd/... ./internal/...

lint:
	golangci-lint run

# Run server with embedded UI. Open http://localhost:3548 (builds binary first).
run: build
	./bin/$(BINARY) start --foreground

# Run Go API (3548) + Vite dev server (5173). Open http://localhost:5173. Requires Node.
run-dev:
	@trap 'kill $$PID 2>/dev/null || true' EXIT INT TERM; \
		go run ./cmd/agentary start --foreground --dev --port=3548 & PID=$$!; \
		sleep 2; \
		cd web && npm run dev

# Build with version from git. Override: VERSION=v1.0.0 make release
release: build
	@echo "Built bin/$(BINARY) (version $(VERSION))"

smoke: test lint
	@echo "Smoke check passed."

fmt:
	gofmt -w ./cmd ./internal

docker-build:
	docker build -t $(BINARY):dev .
