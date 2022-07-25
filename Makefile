.PHONY: build run lint test test-e2e build-wasm

lint:
	golangci-lint run --timeout 5m ./...

build:
	go build -o portal ./cmd/portal/*.go 

build-wasm:
	GOOS=js GOARCH=wasm go build -o portal.wasm ./wasm/main.go

run: build
	./portal -p 8080

test:
	go test -v -race -covermode=atomic -coverprofile=coverage.out -failfast -short ./...

test-e2e:
	docker build --tag rendezvous:latest .
	go test -v -race -covermode=atomic -coverprofile=coverage.out -failfast ./...
