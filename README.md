# ocg — OpenCode Go Usage (macOS Menu Bar)

Menu bar app that monitors your [OpenCode Go](https://opencode.ai) plan usage. A colour-coded circular icon in the menu bar reflects your monthly usage level; click to see full details. Auto-refreshes every 15 minutes.

## Quick Start

```bash
make app                          # build → OCGTool.app
cp -R OCGTool.app ~/OCGTool.app   # copy out of the source tree
open ~/OCGTool.app                # launch
```

> If `open` from the project folder doesn't stick, copy the `.app` to `~/` or
> `/Applications` first — LaunchServices can cache a stale registration.

## Menu

```
[🟢 coloured circle]              ← menu bar icon, colour shifts with usage
  OpenCode Go Usage
  Updated 14:32
  ──────────────
  Rolling    2%   🟢   2h 1m
  Weekly    24%   🟢   18h
  Monthly   63%   🟡   6d 11h
  ──────────────
  Refresh Now
  Set Cookie…
  Set Workspace ID…
  ──────────────
  Quit
```

The menu bar icon is a circular gauge: green (<50%), yellow (50–84%),
red (≥85%), grey (unconfigured/error). Each meter line has a matching
status dot emoji.

## Setup (one-time)

1. Open [opencode.ai/workspace](https://opencode.ai/workspace) and log in with GitHub/Google
2. DevTools → Application → Cookies → `opencode.ai` → copy the `auth` value
3. Click **Set Cookie…** in the menu bar and paste it (with or without the `auth=` prefix)

The cookie is valid for 1 year. Config stored at `~/.config/ocg/config.json`.

## Build

Requires Go 1.22+ and Xcode Command Line Tools (for CGO).

```bash
make        # builds OCGTool.app
make run    # builds and opens the app
make build  # plain binary
```

## How it works

Fetches `https://opencode.ai/workspace/<id>/go` with the saved cookie and parses
usage meter data from the embedded page state. One HTTP call, no headless browser.
The menu bar icon is a procedurally generated PNG that recolours based on usage.
