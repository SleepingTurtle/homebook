.PHONY: build run dev watch docker-up docker-down backup clean seed repopulate

# Build the binary
build:
	go build -o homebooks ./cmd/server

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
