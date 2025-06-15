.PHONY: help dev build clean types backend frontend docker

# Default target
help:
	@echo "Curator Development Commands"
	@echo ""
	@echo "Setup:"
	@echo "  setup      - Install all dependencies"
	@echo "  types      - Generate TypeScript types from schemas"
	@echo ""
	@echo "Development:"
	@echo "  dev        - Start development servers (backend + frontend)"
	@echo "  backend    - Start backend development server"
	@echo "  frontend   - Start frontend development server"
	@echo ""
	@echo "Build:"
	@echo "  build      - Build everything for production"
	@echo "  clean      - Clean build artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  docker         - Build and start all services with Docker Compose"
	@echo "  docker-build   - Build Docker images"
	@echo "  docker-backend - Start only backend and ollama services"
	@echo "  docker-frontend - Start only frontend service"

# Setup commands
setup:
	@echo "Installing Go dependencies..."
	go mod tidy
	@echo "Installing frontend dependencies..."
	cd frontend && npm install
	@echo "Installing build script dependencies..."
	cd scripts/build && npm install
	@echo "Generating types..."
	$(MAKE) types

types:
	@echo "Generating TypeScript types from JSON schemas..."
	cd scripts/build && node generate-types.js

# Development commands
dev:
	@echo "Starting development servers..."
	@($(MAKE) backend &) && ($(MAKE) frontend &) && wait

backend:
	@echo "Starting backend development server..."
	cd backend && go run cmd/curator/main.go

frontend:
	@echo "Starting frontend development server..."
	cd frontend && npm run dev

# Build commands
build: types
	@echo "Building backend..."
	go build -o bin/curator ./cmd/curator
	@echo "Building frontend..."
	cd frontend && npm run build

clean:
	@echo "Cleaning build artifacts..."
	cd backend
	rm -rf bin/
	rm -rf frontend/dist/
	rm -rf shared/types/
	go clean

# Docker commands
docker:
	docker-compose up --build

docker-build:
	docker-compose build

docker-backend:
	docker-compose up --build curator-backend

docker-frontend:
	docker-compose up --build curator-frontend