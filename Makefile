.PHONY: run dev build test tidy fmt frontend-install frontend-start \
	frontend-build build-all docker-build docker-up docker-down

# Backend on :8080
run:
	go run ./cmd/server

# Hot reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air

build:
	go build -o bin/server ./cmd/server

test:
	go test ./... -count=1

tidy:
	go mod tidy

fmt:
	gofmt -w .

frontend-install:
	cd frontend && npm install

# Frontend dev on :3000 (proxy -> :8080)
frontend-start:
	cd frontend && npm start

# Production build of the frontend (-> frontend/dist)
frontend-build:
	cd frontend && npm install && npm run build

# Single-binary local run: build frontend, then serve it from Go on :8080
build-all: frontend-build build
	STATIC_DIR=frontend/dist ./bin/server

# --- Docker ---
docker-build:
	docker build -t clever-dashboard:latest .

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down
