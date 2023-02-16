.PHONY: serve lint test test-e2e build-wasm image

LINKER_FLAGS = '-s -X main.version=${PORTAL_VERSION}'

lint:
	golangci-lint run --timeout 5m ./...

build: 
	go build -o portal-bin ./cmd/portal/

build-production:
	CGO=0 go build -ldflags=${LINKER_FLAGS} -o portal ./cmd/portal/

image: 
	docker build --tag rendezvous:latest .

serve: image
	docker run -dp 8080:8080 rendezvous:latest

build-wasm:
	GOOS=js GOARCH=wasm go build -o portal.wasm ./cmd/wasm/main.go

test:
	go test -v -race -covermode=atomic -coverprofile=coverage.out -failfast -short ./...

test-e2e: image
	go test -v -race -covermode=atomic -coverprofile=coverage.out -failfast ./...
