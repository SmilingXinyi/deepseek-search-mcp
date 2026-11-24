# deepseek-search-mcp

MCP server that exposes the DeepSeek Search API and Fetch API as two read-only tools. The server is a Go binary scaffolded with `mcp-helper` 2.4.0 and the official Go MCP SDK.

- `deepseek_search` searches the web and returns ranked results.
- `deepseek_fetch` fetches one page and returns its title plus Markdown.

Upstream reference: [DeepSeek Search & Fetch](https://api-docs.deepseek.com/guides/search/).

## Requirements

- Go 1.25 or newer
- A DeepSeek API key in `DEEPSEEK_API_KEY`

Search and Fetch are billed by DeepSeek per successful request. Failed requests, empty search results, and timeouts are not billed. This server does not call the live API from tests.

## Configure

```bash
export DEEPSEEK_API_KEY=sk-...
# optional; default is https://api.deepseek.com
export DEEPSEEK_BASE_URL=https://api.deepseek.com
```

Copy `.env.example` when you want the same variables in a file. The key is never written into the repository.

HTTP transport also reads:

| Variable | Default | Purpose |
| --- | --- | --- |
| `MCP_HOST` | `127.0.0.1` | Bind host |
| `MCP_PORT` | `3000` | Bind port |
| `MCP_AUTH_TOKEN` | empty | When set, `POST /mcp` requires `Authorization: Bearer <token>` |
| `MCP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `MCP_LOG_DIR` | `./logs` | Rotating log directory |
| `MCP_LOG_FILE` | enabled | Set to `0` to disable the log file |

Logs go to stderr for the stdio binary so they do not corrupt the MCP stream.

## Run

```bash
go mod tidy
make build
```

`make build` writes `bin/deepseek-search-mcp` (stdio) and `bin/deepseek-search-mcp-http` (Streamable HTTP).

```bash
DEEPSEEK_API_KEY=sk-... make run-stdio
DEEPSEEK_API_KEY=sk-... make run-http
```

Streamable HTTP listens on `http://127.0.0.1:3000/mcp`.

### Cursor

Stdio:

```json
{
  "mcpServers": {
    "deepseek-search": {
      "command": "/absolute/path/to/bin/deepseek-search-mcp",
      "env": {
        "DEEPSEEK_API_KEY": "sk-..."
      }
    }
  }
}
```

Streamable HTTP, after `make run-http`:

```json
{
  "mcpServers": {
    "deepseek-search": {
      "url": "http://127.0.0.1:3000/mcp"
    }
  }
}
```

When `MCP_AUTH_TOKEN` is set, send `Authorization: Bearer <token>` on that URL.

## Tools

Both tools are read-only. Call failures are returned as MCP tool errors. The message includes the HTTP status meaning, for example `401 missing or invalid API key` or `429 rate limit exceeded`. The server retries HTTP 408, 429, and 5xx, and it honors `Retry-After`. Other 4xx responses are not retried. Input is validated locally before the request is sent.

### deepseek_search

`POST /v1/tools/search`

| Parameter | Required | Default | Notes |
| --- | --- | --- | --- |
| `query` | yes | | 1–128 Unicode characters. Not all whitespace. Operators such as `site:`, `-site:`, `lang:`, `after:`, and `before:` are forwarded unchanged. |
| `count` | no | 10 | Integer from 1 to 50. |
| `ori_question` | no | | Original user question, 1–128 characters. |
| `localization` | no | | IP address used to localize results. Send it only when the user provided it. |
| `need_content` | no | false | Include page content on each result. |
| `max_content_chars` | no | 4000 | Local cap on each `content` field when `need_content` is true. `0` disables the cap. Not sent upstream. |

An empty `results` array is success and the text result is `No results found...`. Truncated content ends with `[... truncated at N characters ...]`.

### deepseek_fetch

`POST /v1/tools/fetch`

| Parameter | Required | Default | Notes |
| --- | --- | --- | --- |
| `url` | yes | | Absolute `http` or `https` URL, at most 8192 characters, with a host and no username or password. |
| `max_chars` | no | 0 | Local cap on Markdown content. `0` returns the full page. Not sent upstream. |

The structured result includes `title`, `content`, `truncated`, and `original_chars`.

## Layout

| Path | Responsibility |
| --- | --- |
| `cmd/stdio` | Stdio process entry |
| `cmd/http` | Streamable HTTP process entry |
| `internal/server` | MCP server and tool registration |
| `internal/deepseek` | DeepSeek HTTP client, validation, and formatting |
| `internal/logger` | stderr plus rotating file logs |

## Release

Pushing a `v*` tag builds one archive per platform and publishes them on a GitHub Release. Each archive is a single package of this server: the stdio binary, the Streamable HTTP binary, `README.md`, and `.env.example`. Windows archives are zip files; the others are tar.gz. `checksums.txt` lists SHA-256 digests.

```bash
git tag v0.1.0
git push origin v0.1.0
```

Archive names look like `deepseek-search-mcp_v0.1.0_linux_amd64.tar.gz`. Platforms are linux, darwin, and windows, each for amd64 and arm64.

Build the same archives locally:

```bash
make release
```

## Docker

Build the HTTP image, then run Compose with `IMAGE_NAME` and `DEEPSEEK_API_KEY` set:

```bash
docker build -t deepseek-search-mcp:latest .
docker compose up
```
