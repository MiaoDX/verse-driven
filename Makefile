.PHONY: build test lint vet staticcheck tidy clean all

BINARY := scripture-mcp
CMD := ./cmd/scripture-mcp
BUILD_DIR := bin

all: lint test build

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD)

test:
	go test ./...

vet:
	go vet ./...

staticcheck:
	@command -v staticcheck >/dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...

lint: vet staticcheck

tidy:
	go mod tidy

clean:
	rm -rf $(BUILD_DIR)
