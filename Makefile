.PHONY: help build install lint test test-component clean snapshot

help:
	@echo "make help           Show this help message"
	@echo "make build          Build the tk binary"
	@echo "make install        Install tk to GOPATH/bin"
	@echo "make lint           Run golangci-lint (via Docker)"
	@echo "make test           Run unit tests"
	@echo "make test-component Run component tests against the real binary"
	@echo "make snapshot       Build release snapshot (local test)"
	@echo "make clean          Remove build artifacts"

build:
	go build -o bin/tk ./cmd/tk

install:
	go install ./cmd/tk

lint:
	docker run --rm -v "$$(pwd)":/app -w /app golangci/golangci-lint:v2.11.4 golangci-lint run -v --fix ./...

test:
	go test -race -v ./...

test-component: build
	TK_BINARY=$(CURDIR)/bin/tk go test -race -v ./test/component/

clean:
	rm -rf bin/ dist/

snapshot:
	goreleaser release --snapshot --clean
