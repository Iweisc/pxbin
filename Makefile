.PHONY: build run dev test lint migrate clean frontend-dev frontend-build

BIN := bin/pxbin

build:
	go build -o $(BIN) ./cmd/pxbin

run: build
	./$(BIN)

dev:
	go run ./cmd/pxbin

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

migrate:
	go run ./cmd/pxbin -migrate

clean:
	rm -rf bin/ dist/

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build
