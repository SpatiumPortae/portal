.PHONY: build run lint test test-e2e build-wasm

LINKER_FLAGS = '-s -X main.version=${PORTAL_VERSION}'

lint:
	golangci-lint run --timeout 5m ./...

build:
	go build -o portal ./cmd/portal/

build-production:
	CGO=0 go build -ldflags=${LINKER_FLAGS} -o portal ./cmd/portal/

build-wasm:
	GOOS=js GOARCH=wasm go build -o portal.wasm ./cmd/wasm/main.go

run: build
	./portal -p 8080

test:
	go test -v -race -covermode=atomic -coverprofile=coverage.out -failfast -short ./...

test-e2e:
	docker build --tag rendezvous:latest .
	go test -v -race -covermode=atomic -coverprofile=coverage.out -failfast ./...
