.PHONY: build run dev test lint clean docker-up docker-down

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

run: build
	./bin/server

dev:
	go run ./cmd/server

test:
	go test ./... -v -count=1

lint:
	go vet ./...

clean:
	rm -rf bin/

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f
