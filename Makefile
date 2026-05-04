.PHONY: build test lint vet staticcheck tidy clean all packs verify-packs

BINARY := scripture-mcp
CMD := ./cmd/scripture-mcp
BUILD_DIR := bin
STATICCHECK_VERSION := v0.6.1

all: lint verify-packs test build

# Rebuild bundled packs from upstream sources. Run after upstream
# regenerations or whenever the JSONL format changes. Requires Python 3.11+;
# opencc-python-reimplemented is required for the zh-Hans Dao/Sutra targets.
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
	@command -v staticcheck >/dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	staticcheck ./...

lint: vet staticcheck

tidy:
	go mod tidy

clean:
	rm -rf $(BUILD_DIR)
