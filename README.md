# ocg — OpenCode Go Usage CLI

Query your [OpenCode Go](https://opencode.ai) plan usage from the terminal. No browser needed after initial setup.

## Usage

```
ocg         # show usage
ocg --json  # JSON output
```

### Rolling / Weekly / Monthly

```
OpenCode Go Usage
  Plan: Go

  Rolling:   2% used  (resets in 2h 1m)
  Rolling ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  2%
  …
```

### JSON

```json
{
  "rolling":   { "percent": 2, "reset_in_sec": 7746,  "status": "ok" },
  "weekly":    { "percent": 24,"reset_in_sec": 64856, "status": "ok" },
  "monthly":   { "percent": 63,"reset_in_sec": 558272,"status": "ok" },
  "plan": "Go",
  "fetched_at": "2026-06-21T05:59:53Z"
}
```

## Setup (one-time)

```
ocg cookie '<auth-cookie-value>'   # paste auth cookie from browser DevTools
ocg workspace <id>                 # set workspace ID (default: your workspace)
```

To get the cookie:

1. Open [opencode.ai/workspace](https://opencode.ai/workspace) and log in
2. DevTools → Application → Cookies → `opencode.ai` → copy `auth` value
3. `ocg cookie 'auth=…'`

The cookie is valid for 1 year.

## Build

```
go build -o ocg .
```

Cross-compile for Linux:

```
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ocg-linux-amd64 .
```

## How it works

Fetches `https://opencode.ai/workspace/<id>/go` with the saved cookie and parses usage meter data from the embedded page state. One HTTP call, no headless browser.
