.PHONY: help build install lint test clean snapshot

help:
	@echo "make help     Show this help message"
	@echo "make build    Build the tk binary"
	@echo "make install  Install tk to GOPATH/bin"
	@echo "make lint     Run golangci-lint"
	@echo "make test     Run tests"
	@echo "make snapshot Build release snapshot (local test)"
	@echo "make clean    Remove build artifacts"

build:
	go build -o bin/tk ./cmd/tk

install:
	go install ./cmd/tk

lint:
	golangci-lint run ./...

test:
	go test -race -v ./...

clean:
	rm -rf bin/ dist/

snapshot:
	goreleaser release --snapshot --clean
