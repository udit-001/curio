.PHONY: build install clean run test fmt check tidy

build:
	mkdir -p bin
	go build -o bin/curio ./cmd/curio

install:
	go install ./cmd/curio

clean:
	rm -f bin/curio
	go clean

run:
	go run ./cmd/curio

test:
	go test ./...

fmt:
	gofmt -s -w .

check:
	test -z "$$(gofmt -l .)"
	go vet ./...

tidy:
	go mod tidy
