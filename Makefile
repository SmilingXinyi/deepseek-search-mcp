NAME := deepseek-search-mcp
BIN_STDIO := bin/$(NAME)
BIN_HTTP := bin/$(NAME)-http

.PHONY: build build-stdio build-http run run-stdio run-http tidy clean

build: build-stdio build-http

build-stdio:
	go build -o $(BIN_STDIO) ./cmd/stdio

build-http:
	go build -o $(BIN_HTTP) ./cmd/http

run: run-stdio

run-stdio: build-stdio
	$(BIN_STDIO)

run-http: build-http
	$(BIN_HTTP)

tidy:
	go mod tidy

clean:
	rm -rf bin logs
