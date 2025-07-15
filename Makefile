.PHONY: build dependencies test test-race test-cover test-e2e lint lint-ci lintfile tidy man clean install help

help:
	@echo "🧁 Available:"
	@echo "  build       - Build the sift binary"
	@echo "  dependencies- Download all dependencies"
	@echo "  test        - Run all tests"
	@echo "  test-race   - Run tests with race detection"
	@echo "  test-cover  - Run tests with coverage report"
	@echo "  test-e2e    - Run end-to-end tests"
	@echo "  lint        - Run linters (go vet, gofmt)"
	@echo "  lint-ci     - Run golangci-lint (CI-grade linting)"
	@echo "  lintfile    - Run golangci-lint on a specific file"
	@echo "  tidy        - Clean up dependencies"
	@echo "  man         - Generate man pages"
	@echo "  clean       - Clean build artifacts"
	@echo "  install     - Install sift to GOPATH/bin"

build:
	$(info 🧁 building sift )
	go build -ldflags="-s -w" -o sift ./cmd/sift

dependencies:
	$(info 🧁 download dependencies  )
	go mod download
	go mod verify

test:
	$(info 🧁 running tests )
	go test -v ./...

test-race:
	$(info 🧁 running tests with race detection  )
	go test -v -race ./...

test-cover:
	$(info 🧁 running tests with coverage  )
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-e2e:
	$(info 🧁 running end-to-end tests  )
	go test -v --tags=e2e ./cmd/sift/

lint:
	$(info 🧁 checking code quality  )
	go vet ./...
	gofmt -l .
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Files need formatting:"; \
		gofmt -l .; \
		exit 1; \
	fi

lint-ci:
	$(info 🧁 running golangci-lint  )
	golangci-lint run

lintfile:
	$(info 🧁 running golangci-lint on $(FILE) )
	@if [ -z "$(FILE)" ]; then \
		echo "Usage: make lintfile FILE=<file_path>"; \
		echo "Example: make lintfile FILE=main.go"; \
		exit 1; \
	fi
	@if [ ! -f "$(FILE)" ]; then \
		echo "Error: File '$(FILE)' does not exist"; \
		exit 1; \
	fi

	@golangci-lint run "$(FILE)"

tidy:
	$(info 🧁 clean up dependencies  )
	go mod tidy

man:
	$(info 🧁 building man pages  )
	go run ./cmd/sift man

clean:
	$(info 🧁 cleaning build artifacts  )
	rm -f sift
	rm -f coverage.out coverage.html
	rm -rf dist/

install: build
	$(info 🧁 installing sift 🧁 )
	go install ./cmd/sift

cyclo:
	$(info 🧁 Assessing cyclomatic complexity)
	gocyclo -top 10 . | grep "^[2-9][0-9]"