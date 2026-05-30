.PHONY: run dev build test tidy fmt frontend-install frontend-start

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
