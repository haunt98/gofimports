.PHONY: all test test-color coverage coverage-cli coverate-html lint format build clean docs

all:
	go mod tidy
	$(MAKE) test-color
	$(MAKE) lint
	$(MAKE) format
	$(MAKE) build
	$(MAKE) clean

test:
	go test -race -failfast ./...

test-color:
	go install github.com/haunt98/go-test-color@latest
	go-test-color -race -failfast ./...

coverage:
	go test -coverprofile=coverage.out ./...

coverage-cli:
	$(MAKE) coverage
	go tool cover -func=coverage.out

coverage-html:
	$(MAKE) coverage
	go tool cover -html=coverage.out

lint:
	golangci-lint run ./...

format:
	$(MAKE) build
	go install mvdan.cc/gofumpt@latest
	./gofimports -w --company github.com/make-go-great,github.com/haunt98 .
	gofumpt -w -extra .
	$(MAKE) clean

build:
	$(MAKE) clean
	go build ./cmd/gofimports

clean:
	rm -rf ./gofimports

docs:
	go install go101.org/golds@latest
	golds -theme=dark ./...
