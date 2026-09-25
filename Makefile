include tools/versions.env

GO ?= go
GOWORK ?= off
export GOWORK

BIN_DIR := $(CURDIR)/.tmp/bin
export PATH := $(BIN_DIR):$(PATH)

.PHONY: help tools fmt-check vet lint test test-race vuln mod-verify \
	quickstart secrets consumer-published verify

help:
	@echo "Targets:"
	@echo "  fmt-check           fail when tracked Go files need gofmt"
	@echo "  vet                 run go vet"
	@echo "  lint                run golangci-lint"
	@echo "  test                run unit and example tests"
	@echo "  test-race           run tests with the race detector"
	@echo "  vuln                run govulncheck"
	@echo "  mod-verify          verify downloaded module checksums"
	@echo "  quickstart          build and test the quickstart"
	@echo "  secrets             scan git history and worktree with Gitleaks"
	@echo "  consumer-published  test the published module without replace"
	@echo "  verify              run the complete local quality gate"

tools: $(BIN_DIR)/golangci-lint $(BIN_DIR)/govulncheck $(BIN_DIR)/gitleaks

$(BIN_DIR)/golangci-lint:
	@mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(BIN_DIR)/govulncheck:
	@mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) $(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

$(BIN_DIR)/gitleaks:
	@mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) $(GO) install github.com/zricethezav/gitleaks/v8@$(GITLEAKS_VERSION)

fmt-check:
	@files="$$(git ls-files '*.go')"; \
	out="$$(gofmt -l $$files)"; \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	git diff --check

vet:
	$(GO) vet ./...

lint: $(BIN_DIR)/golangci-lint
	$(BIN_DIR)/golangci-lint run ./...

test:
	$(GO) test -count=1 ./...

test-race:
	$(GO) test -race -count=1 ./...

vuln: $(BIN_DIR)/govulncheck
	$(BIN_DIR)/govulncheck ./...

mod-verify:
	$(GO) mod verify

quickstart:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/authbase-quickstart ./examples/quickstart
	$(GO) test -count=1 ./examples/quickstart

secrets: $(BIN_DIR)/gitleaks
	./tools/check-secrets.sh

consumer-published:
	./tools/verify-published-module.sh $(AUTHBASE_VERSION)

verify: fmt-check mod-verify vet test test-race lint vuln quickstart secrets
	@echo "verify OK"
