FROM golang:1.25-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/http

FROM debian:bookworm-slim

RUN useradd --create-home --uid 1000 mcp \
    && mkdir -p /home/mcp/logs \
    && chown -R mcp:mcp /home/mcp
WORKDIR /home/mcp
COPY --from=build /out/server /usr/local/bin/mcp-server
USER mcp
ENV MCP_LOG_DIR=/home/mcp/logs
ENV MCP_HOST=0.0.0.0
ENV MCP_PORT=3000
EXPOSE 3000
ENTRYPOINT ["mcp-server"]
