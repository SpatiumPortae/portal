.PHONY: build run lint test test-e2e
lint:
	golangci-lint run --timeout 5m ./...

build:
	go build -o portal ./cmd/portal/*.go 

run:
	./portal -p 8080

test:
	go test -v -race -covermode=atomic -coverprofile=coverage.out -failfast -short ./...

test-e2e:
	docker build --tag rendezvous:latest .
	go test -v -race -covermode=atomic -coverprofile=coverage.out -failfast ./...
