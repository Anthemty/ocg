# ocg — OpenCode Go Usage (macOS Menu Bar)

Menu bar app that monitors your [OpenCode Go](https://opencode.ai) plan usage. Sits in the menu bar and auto-refreshes every 15 minutes.

## Quick Start

```bash
make app       # build → OCGTool.app
open OCGTool.app
```

Or just the binary (shows in Dock too without the .app wrapper):

```bash
go build -o ocg .
./ocg
```

## Menu

```
[●]                   ← menu bar icon
  ──────────────
  Plan: Go            ← usage data (updated every 15 min)
  ──────────────
  Rolling: 2% used (resets in 2h)
  Weekly: 24% used (resets in 18h)
  Monthly: 63% used (resets in 6d)
  ──────────────
  Refresh Now         ← manual refresh
  Set Cookie...       ← paste auth cookie from browser
  Set Workspace ID... ← change workspace
  ──────────────
  Quit
```

## Setup (one-time)

1. Open [opencode.ai/workspace](https://opencode.ai/workspace) and log in with GitHub/Google
2. DevTools → Application → Cookies → `opencode.ai` → copy the `auth` value
3. Click **Set Cookie...** in the menu bar and paste it (with or without the `auth=` prefix)

The cookie is valid for 1 year.

App config is stored at `~/.config/ocg/config.json`.

## Build

Requires Go 1.22+ and Xcode Command Line Tools (for CGO — calls macOS native APIs).

```bash
make        # builds OCGTool.app
make run    # builds and opens the app
make build  # plain binary (no .app wrapper)
```

## How it works

Fetches `https://opencode.ai/workspace/<id>/go` with the saved cookie and parses usage meter data from the embedded page state. One HTTP call, no headless browser.
