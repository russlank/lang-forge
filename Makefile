GO ?= /usr/local/go/bin/go
DOTNET ?= dotnet
CXX ?= g++

APP_NAME := lang-forge
CMD_PATH := ./cmd/lang-forge
DIST_DIR := dist
MODULE := github.com/russlank/lang-forge

DOCKERFILE ?= Dockerfile
IMAGE_REPO ?= lang-forge
IMAGE_TAG ?= $(VERSION)
IMAGE ?= $(IMAGE_REPO):$(IMAGE_TAG)
REPO_URL ?= $(shell git config --get remote.origin.url 2>/dev/null || echo unknown)

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)

GOFLAGS := -trimpath
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.BuildDate=$(DATE) \
	-X $(MODULE)/internal/version.Branch=$(BRANCH)

.PHONY: all ci fmt fmt-check vet test test-race fuzz-smoke golden-stability \
	vulncheck tidy build install \
	dist linux-amd64 linux-arm64 darwin-arm64 darwin-amd64 windows-amd64 \
	examples-generate examples-run examples-test examples-cleanliness \
	examples-parity examples-target-env-smoke examples-testdata examples-templates examples-clean \
	docker-build docker-smoke docker-push image-tags clean

all: fmt vet test build

ci: fmt-check vet test-race fuzz-smoke golden-stability build examples-test

fmt:
	$(GO) fmt ./...

fmt-check:
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
	  echo "The following files are not gofmt-formatted:"; \
	  echo "$$unformatted"; \
	  echo "Run: make fmt"; \
	  exit 1; \
	fi

vet:
	$(GO) vet ./...

test:
	$(GO) test -count=1 ./...

test-race:
	CGO_ENABLED=1 $(GO) test -race -count=1 ./...

fuzz-smoke:
	$(GO) test -run '^$$' -fuzz=FuzzParseRegexSmoke -fuzztime=1s ./internal/lex
	$(GO) test -run '^$$' -fuzz=FuzzParseCombinedSmoke -fuzztime=1s ./internal/spec

golden-stability:
	GO=$(GO) sh scripts/check-golden-stability.sh

vulncheck:
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest
	$$($(GO) env GOPATH)/bin/govulncheck ./...

tidy:
	$(GO) mod tidy

build:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST_DIR)/$(APP_NAME) $(CMD_PATH)

install: build
	install -m 0755 $(DIST_DIR)/$(APP_NAME) $${PREFIX:-/usr/local}/bin/$(APP_NAME)

linux-amd64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 $(CMD_PATH)

linux-arm64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 $(CMD_PATH)

darwin-arm64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 $(CMD_PATH)

darwin-amd64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 $(CMD_PATH)

windows-amd64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe $(CMD_PATH)

dist: linux-amd64 linux-arm64 darwin-arm64 darwin-amd64 windows-amd64
	cp scripts/install-lang-forge.sh $(DIST_DIR)/install-lang-forge.sh
	chmod 0755 $(DIST_DIR)/install-lang-forge.sh
	cd $(DIST_DIR) && sha256sum $(APP_NAME)-linux-* $(APP_NAME)-darwin-* $(APP_NAME)-windows-* install-lang-forge.sh > SHA256SUMS

examples-generate:
	$(MAKE) -C examples/go/calc GO=$(GO) generate
	$(MAKE) -C examples/go/datakeeper GO=$(GO) generate
	$(MAKE) -C examples/go/draw GO=$(GO) generate
	$(MAKE) -C examples/go/parser-recovery GO=$(GO) generate
	$(MAKE) -C examples/go/vehicle-report GO=$(GO) generate
	$(MAKE) -C examples/csharp/calc GO=$(GO) DOTNET=$(DOTNET) generate
	$(MAKE) -C examples/csharp/datakeeper GO=$(GO) DOTNET=$(DOTNET) generate
	$(MAKE) -C examples/csharp/draw GO=$(GO) DOTNET=$(DOTNET) generate
	$(MAKE) -C examples/csharp/vehicle-report GO=$(GO) DOTNET=$(DOTNET) generate
	$(MAKE) -C examples/c/calc GO=$(GO) generate
	$(MAKE) -C examples/c/datakeeper GO=$(GO) generate
	$(MAKE) -C examples/c/draw GO=$(GO) generate
	$(MAKE) -C examples/c/vehicle-report GO=$(GO) generate
	$(MAKE) -C examples/cpp/calc GO=$(GO) CXX=$(CXX) generate
	$(MAKE) -C examples/cpp/datakeeper GO=$(GO) CXX=$(CXX) generate
	$(MAKE) -C examples/cpp/draw GO=$(GO) CXX=$(CXX) generate
	$(MAKE) -C examples/cpp/vehicle-report GO=$(GO) CXX=$(CXX) generate
	$(MAKE) -C examples/templates/go/mini-compiler GO=$(GO) generate
	$(MAKE) -C examples/templates/csharp/mini-compiler GO=$(GO) DOTNET=$(DOTNET) generate
	$(MAKE) -C examples/templates/c/mini-compiler GO=$(GO) generate
	$(MAKE) -C examples/templates/cpp/mini-compiler GO=$(GO) CXX=$(CXX) generate

examples-run:
	$(MAKE) -C examples/go/calc GO=$(GO) run
	$(MAKE) -C examples/go/datakeeper GO=$(GO) run
	$(MAKE) -C examples/go/draw GO=$(GO) run
	$(MAKE) -C examples/go/parser-recovery GO=$(GO) run
	$(MAKE) -C examples/go/vehicle-report GO=$(GO) run
	$(MAKE) -C examples/csharp/calc GO=$(GO) DOTNET=$(DOTNET) run
	$(MAKE) -C examples/csharp/datakeeper GO=$(GO) DOTNET=$(DOTNET) run
	$(MAKE) -C examples/csharp/draw GO=$(GO) DOTNET=$(DOTNET) run
	$(MAKE) -C examples/csharp/vehicle-report GO=$(GO) DOTNET=$(DOTNET) run
	$(MAKE) -C examples/c/calc GO=$(GO) run
	$(MAKE) -C examples/c/datakeeper GO=$(GO) run
	$(MAKE) -C examples/c/draw GO=$(GO) run
	$(MAKE) -C examples/c/vehicle-report GO=$(GO) run
	$(MAKE) -C examples/cpp/calc GO=$(GO) CXX=$(CXX) run
	$(MAKE) -C examples/cpp/datakeeper GO=$(GO) CXX=$(CXX) run
	$(MAKE) -C examples/cpp/draw GO=$(GO) CXX=$(CXX) run
	$(MAKE) -C examples/cpp/vehicle-report GO=$(GO) CXX=$(CXX) run
	$(MAKE) -C examples/templates/go/mini-compiler GO=$(GO) run
	$(MAKE) -C examples/templates/csharp/mini-compiler GO=$(GO) DOTNET=$(DOTNET) run
	$(MAKE) -C examples/templates/c/mini-compiler GO=$(GO) run
	$(MAKE) -C examples/templates/cpp/mini-compiler GO=$(GO) CXX=$(CXX) run

examples-test:
	$(MAKE) examples-cleanliness
	$(MAKE) examples-parity
	$(MAKE) examples-target-env-smoke
	$(MAKE) examples-testdata
	$(MAKE) examples-templates
	$(MAKE) -C examples/parser-algorithms GO=$(GO) test
	$(MAKE) -C examples/go/calc GO=$(GO) test
	$(MAKE) -C examples/go/datakeeper GO=$(GO) test
	$(MAKE) -C examples/go/draw GO=$(GO) test
	$(MAKE) -C examples/go/parser-recovery GO=$(GO) test
	$(MAKE) -C examples/go/vehicle-report GO=$(GO) test
	$(MAKE) -C examples/csharp/calc GO=$(GO) DOTNET=$(DOTNET) test
	$(MAKE) -C examples/csharp/datakeeper GO=$(GO) DOTNET=$(DOTNET) test
	$(MAKE) -C examples/csharp/draw GO=$(GO) DOTNET=$(DOTNET) test
	$(MAKE) -C examples/csharp/vehicle-report GO=$(GO) DOTNET=$(DOTNET) test
	$(MAKE) -C examples/c/calc GO=$(GO) test
	$(MAKE) -C examples/c/datakeeper GO=$(GO) test
	$(MAKE) -C examples/c/draw GO=$(GO) test
	$(MAKE) -C examples/c/vehicle-report GO=$(GO) test
	$(MAKE) -C examples/cpp/calc GO=$(GO) CXX=$(CXX) test
	$(MAKE) -C examples/cpp/datakeeper GO=$(GO) CXX=$(CXX) test
	$(MAKE) -C examples/cpp/draw GO=$(GO) CXX=$(CXX) test
	$(MAKE) -C examples/cpp/vehicle-report GO=$(GO) CXX=$(CXX) test

examples-cleanliness:
	sh scripts/check-example-cleanliness.sh

examples-parity:
	$(GO) run ./cmd/check-example-spec-parity

examples-target-env-smoke:
	sh scripts/check-example-target-variable.sh

examples-testdata:
	GO=$(GO) sh scripts/check-example-testdata.sh

examples-templates:
	$(MAKE) -C examples/templates/go/mini-compiler GO=$(GO) test
	$(MAKE) -C examples/templates/csharp/mini-compiler GO=$(GO) DOTNET=$(DOTNET) test
	$(MAKE) -C examples/templates/c/mini-compiler GO=$(GO) test
	$(MAKE) -C examples/templates/cpp/mini-compiler GO=$(GO) CXX=$(CXX) test

examples-clean:
	$(MAKE) -C examples/go/calc clean
	$(MAKE) -C examples/go/datakeeper clean
	$(MAKE) -C examples/go/draw clean
	$(MAKE) -C examples/go/parser-recovery clean
	$(MAKE) -C examples/go/vehicle-report clean
	$(MAKE) -C examples/csharp/calc clean
	$(MAKE) -C examples/csharp/datakeeper clean
	$(MAKE) -C examples/csharp/draw clean
	$(MAKE) -C examples/csharp/vehicle-report clean
	$(MAKE) -C examples/c/calc clean
	$(MAKE) -C examples/c/datakeeper clean
	$(MAKE) -C examples/c/draw clean
	$(MAKE) -C examples/c/vehicle-report clean
	$(MAKE) -C examples/cpp/calc clean
	$(MAKE) -C examples/cpp/datakeeper clean
	$(MAKE) -C examples/cpp/draw clean
	$(MAKE) -C examples/cpp/vehicle-report clean
	$(MAKE) -C examples/templates/go/mini-compiler clean
	$(MAKE) -C examples/templates/csharp/mini-compiler clean
	$(MAKE) -C examples/templates/c/mini-compiler clean
	$(MAKE) -C examples/templates/cpp/mini-compiler clean

docker-build:
	docker build \
		--build-arg VERSION="$(VERSION)" \
		--build-arg COMMIT="$(COMMIT)" \
		--build-arg BUILD_DATE="$(DATE)" \
		--build-arg BRANCH="$(BRANCH)" \
		--build-arg GIT_SHA="$(COMMIT)" \
		--build-arg GIT_BRANCH="$(BRANCH)" \
		--build-arg REPO_URL="$(REPO_URL)" \
		--build-arg REPO_TYPE="git" \
		--build-arg CI="false" \
		-f $(DOCKERFILE) \
		-t $(IMAGE) \
		.

docker-smoke: docker-build
	docker run --rm $(IMAGE) version
	docker run --rm -v "$$(pwd):/workspace:ro" -w /workspace $(IMAGE) validate --spec examples/go/calc/calc.lf

docker-push: docker-build
	docker push $(IMAGE)

image-tags:
	./scripts/build-image-tags.sh .tags $(DIST_DIR)/IMAGE_RELEASE_REF.txt

clean:
	rm -rf $(DIST_DIR) .tags
	$(MAKE) examples-clean
