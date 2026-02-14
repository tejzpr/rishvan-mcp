# rishvan-mcp

A stdio MCP server that provides a human-in-the-loop tool (`ask_rishvan`) with a web UI for responding to LLM questions.

## How it works

1. An LLM client calls `ask_rishvan(question, app_name)` via MCP stdio
2. The server creates a pending request in SQLite (scoped to the IDE), starts a web UI on port `56234`, and opens the browser
3. The human sees the question in the React UI and types a response
4. The response is returned to the LLM as the tool result

Multiple IDEs can use the same server binary — each IDE passes `--ide <name>` and gets isolated state.

## Build

Requires Go 1.23+ and Node.js 18+.

```bash
make
```

This builds the frontend, embeds it into the Go binary, and produces `./rishvan-mcp`.

## Usage

```bash
rishvan-mcp --ide <ide-name>
```

The `--ide` flag is **required**. It scopes all DB records and UI state to that IDE instance.

## MCP Configuration

Add to your MCP client config (e.g. Claude Desktop, Windsurf):

```json
{
  "mcpServers": {
    "rishvan": {
      "command": "/path/to/rishvan-mcp",
      "args": ["--ide", "windsurf"]
    }
  }
}
```

## Docker

```bash
docker pull <username>/rishvan-mcp:latest
docker run --rm -it <username>/rishvan-mcp --ide windsurf
```

Images are published to Docker Hub on each GitHub release for `linux/amd64` and `linux/arm64`.

## Tool

### `ask_rishvan`

| Parameter  | Type   | Required | Description |
|------------|--------|----------|-------------|
| `question` | string | yes      | The question or prompt for the human |
| `app_name` | string | yes      | Application/project context name |

**Returns:** The human's text response.

## Data

- Database: `~/.rishvan-mcp/app.db` (SQLite via GORM)
- Web UI: `http://localhost:56234`
