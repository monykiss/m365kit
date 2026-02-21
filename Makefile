BINARY_NAME := kit
BUILD_DIR := ./bin
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/klytics/m365kit/cmd/version.Version=$(VERSION)"

.PHONY: build test lint install clean demo release fmt vet

build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

test:
	go test ./... -v -count=1
	@if [ -d packages/core ] && [ -f packages/core/package.json ]; then \
		cd packages/core && npm test 2>/dev/null || true; \
	fi

lint:
	golangci-lint run ./...

fmt:
	gofmt -w $(GO_FILES)

vet:
	go vet ./...

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to /usr/local/bin/"

clean:
	rm -rf $(BUILD_DIR)
	go clean

demo: build
	@echo "=== M365Kit Demo ==="
	@echo ""
	@echo "--- kit word read sample.docx ---"
	$(BUILD_DIR)/$(BINARY_NAME) word read testdata/sample.docx
	@echo ""
	@echo "--- kit word read sample.docx --json ---"
	$(BUILD_DIR)/$(BINARY_NAME) word read testdata/sample.docx --json
	@echo ""
	@echo "--- kit excel read sample.xlsx --json ---"
	$(BUILD_DIR)/$(BINARY_NAME) excel read testdata/sample.xlsx --json
	@echo ""
	@echo "=== Demo Complete ==="

release:
	goreleaser release --snapshot --clean
