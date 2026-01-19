.PHONY: build run dev watch docker-up docker-down backup clean seed repopulate release tailwind-install tailwind-build tailwind-watch fmt lint setup

# Version info
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -X homebooks/internal/version.Version=$(VERSION) \
           -X homebooks/internal/version.BuildTime=$(BUILD_TIME) \
           -X homebooks/internal/version.GitCommit=$(GIT_COMMIT)

# Build the binary
build:
	go build -ldflags "$(LDFLAGS)" -o homebooks ./cmd/server

# Build optimized release binary
release:
	@echo "Building release $(VERSION)..."
	go build -ldflags "$(LDFLAGS) -s -w" -o homebooks ./cmd/server

# Run the compiled binary
run: build
	./homebooks

# Run in development mode
dev:
	go run ./cmd/server

# Run with hot reload (requires: go install github.com/air-verse/air@latest)
watch:
	air

# Start Docker containers
docker-up:
	docker-compose up -d

# Stop Docker containers
docker-down:
	docker-compose down

# Build Docker image
docker-build:
	docker-compose build

# Backup the database
backup:
	@mkdir -p backups
	@if [ -f ./data/homebooks.db ]; then \
		cp ./data/homebooks.db ./backups/homebooks-$$(date +%Y%m%d-%H%M%S).db; \
		echo "Backup created: backups/homebooks-$$(date +%Y%m%d-%H%M%S).db"; \
	else \
		echo "No database found at ./data/homebooks.db"; \
	fi

# Clean build artifacts
clean:
	rm -f homebooks
	rm -rf data/

# Download dependencies
deps:
	go mod download
	go mod tidy

# Seed the database with sample data
seed:
	@if [ -f ./data/homebooks.db ]; then \
		sqlite3 ./data/homebooks.db < scripts/seed_vendors_expenses.sql; \
		sqlite3 ./data/homebooks.db < scripts/seed_sales.sql; \
		echo "Seed data loaded successfully"; \
	else \
		echo "No database found. Run 'make dev' first to create the database, then run 'make seed'"; \
	fi

# Clean, initialize database, and seed with sample data
repopulate:
	@echo "Cleaning..."
	@rm -f homebooks
	@rm -rf data/
	@echo "Starting server to initialize database..."
	@go run ./cmd/server & SERVER_PID=$$!; \
		sleep 2; \
		echo "Seeding database..."; \
		sqlite3 ./data/homebooks.db < scripts/seed_vendors_expenses.sql; \
		sqlite3 ./data/homebooks.db < scripts/seed_sales.sql; \
		echo "Stopping server..."; \
		kill $$SERVER_PID 2>/dev/null; \
		echo "Done! Database repopulated with seed data."

# Tailwind CSS
TAILWIND_VERSION := v3.4.17

# Download Tailwind CLI binary for current platform
tailwind-install:
	@if [ ! -f ./tailwindcss ]; then \
		echo "Downloading Tailwind CSS CLI..."; \
		ARCH=$$(uname -m); \
		OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		if [ "$$ARCH" = "arm64" ] && [ "$$OS" = "darwin" ]; then \
			curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-macos-arm64; \
			mv tailwindcss-macos-arm64 tailwindcss; \
		elif [ "$$ARCH" = "x86_64" ] && [ "$$OS" = "darwin" ]; then \
			curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-macos-x64; \
			mv tailwindcss-macos-x64 tailwindcss; \
		elif [ "$$ARCH" = "x86_64" ] && [ "$$OS" = "linux" ]; then \
			curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-linux-x64; \
			mv tailwindcss-linux-x64 tailwindcss; \
		elif [ "$$ARCH" = "aarch64" ] && [ "$$OS" = "linux" ]; then \
			curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-linux-arm64; \
			mv tailwindcss-linux-arm64 tailwindcss; \
		else \
			echo "Unsupported platform: $$OS $$ARCH"; \
			exit 1; \
		fi; \
		chmod +x tailwindcss; \
		echo "Tailwind CSS CLI installed successfully"; \
	else \
		echo "Tailwind CSS CLI already installed"; \
	fi

# Build Tailwind CSS
tailwind-build: tailwind-install
	./tailwindcss -i ./web/static/tailwind.css -o ./web/static/tailwind-out.css --minify

# Watch Tailwind CSS for changes
tailwind-watch: tailwind-install
	./tailwindcss -i ./web/static/tailwind.css -o ./web/static/tailwind-out.css --watch

# Format Go code
fmt:
	gofmt -w .

# Lint Go code
lint:
	go vet ./...
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files need formatting:"; \
		gofmt -l .; \
		exit 1; \
	fi

# Setup development environment
setup:
	git config core.hooksPath .githooks
	@echo "Git hooks configured"
