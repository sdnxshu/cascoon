APP_NAME := cascoon
BINARY := bin/$(APP_NAME)
MAIN := cmd/server/main.go

.PHONY: all build run clean test fmt vet tidy docker-build docker-run docker-up docker-down

all: build

build:
	@echo "🔨 Building..."
	go build -o $(BINARY) $(MAIN)

run:
	@echo "🚀 Running..."
	go run $(MAIN)

clean:
	@echo "🧹 Cleaning..."
	rm -rf bin/

test:
	@echo "🧪 Running tests..."
	go test ./...

fmt:
	@echo "🎨 Formatting..."
	go fmt ./...

vet:
	@echo "🔍 Vetting..."
	go vet ./...

tidy:
	@echo "📦 Tidying modules..."
	go mod tidy

docker-build:
	@echo "🔨 Building Docker image..."
	docker build -t $(APP_NAME) .

docker-run:
	@echo "🚀 Running Docker container..."
	docker run -p 8080:8080 $(APP_NAME)

docker-up:
	@echo "🐳 Starting Docker Compose..."
	docker compose up --build -d

docker-up-scale-worker:
	@echo "🐳 Starting Docker Compose with 3 workers..."
	docker compose up --build -d --scale worker=3

docker-down:
	@echo "🐳 Stopping Docker Compose..."
	docker compose down -v

docker-up-dev:
	@echo "🐳 Starting Development Docker Compose..."
	docker compose -f compose.dev.yaml up --build -d

docker-down-dev:
	@echo "🐳 Stopping Development Docker Compose..."
	docker compose -f compose.dev.yaml down -v

sqlc-generate:
	@echo "Generating SQLC code..."
	sqlc generate

sqlc-migrate:
	@echo "Migrating database..."
	