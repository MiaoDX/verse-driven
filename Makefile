.PHONY: build test lint vet staticcheck tidy clean all packs verify-packs

BINARY := scripture-mcp
CMD := ./cmd/scripture-mcp
BUILD_DIR := bin

all: lint verify-packs test build

# Rebuild the bundled packs from upstream sources (KJV from Project
# Gutenberg, 道德经 from Project Gutenberg). Run after upstream regenerations
# or whenever the JSONL format changes. Requires Python 3.11+ and
# opencc-python-reimplemented for the dao pack.
packs:
	python3 scripts/build_packs.py

# Recompute SHA-256 over every bundled verse and compare to the stored
# checksum_sha256. CI runs this as a gate before `go test`.
verify-packs:
	python3 scripts/verify_quotes.py

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
