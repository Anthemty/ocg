# ocg — OpenCode Go Usage (macOS Menu Bar)

Menu bar app that monitors your [OpenCode Go](https://opencode.ai) plan usage. Sits in the menu bar with a live SF Symbol gauge, color-coded percentages, and auto-refreshes every 15 minutes. Built on native macOS AppKit via [darwinkit](https://github.com/progrium/darwinkit).

## Quick Start

```bash
make app       # build → OCGTool.app
open OCGTool.app
```

> If `open OCGTool.app` from the project folder doesn't stick, copy the
> `.app` to `~/` or `/Applications` first — LaunchServices can cache a stale
> registration for bundles living on mounted volumes.

## Menu

```
[⌥ gauge]                         ← menu bar icon (color shifts with usage)
  OpenCode Go Usage              ← semibold header
  Updated 14:32                  ← tertiary label
  ──────────────
  Rolling    2%   resets in 2h 1m   ← label secondary, % green/yellow/red
  Weekly    24%   resets in 18h
  Monthly   63%   resets in 6d 11h
  ──────────────
  Refresh Now
  Set Cookie…
  Set Workspace ID…
  ──────────────
  Quit
```

Percentages are colored with system colors: green (<50%), yellow (50–84%),
red (≥85%). The menu bar gauge symbol swaps between `gauge.low` / `gauge.medium`
/ `gauge.high` / `gauge.with.dots.needle.0percent` based on monthly usage.

## Setup (one-time)

1. Open [opencode.ai/workspace](https://opencode.ai/workspace) and log in with GitHub/Google
2. DevTools → Application → Cookies → `opencode.ai` → copy the `auth` value
3. Click **Set Cookie…** in the menu bar and paste it (with or without the `auth=` prefix)

The cookie is valid for 1 year. Config stored at `~/.config/ocg/config.json`.

## Build

Requires Go 1.22+ and Xcode Command Line Tools (CGO — calls native AppKit).

```bash
make        # builds OCGTool.app
make run    # builds and opens the app
make build  # plain binary
```

## How it works

Fetches `https://opencode.ai/workspace/<id>/go` with the saved cookie and parses
usage meter data from the embedded page state. One HTTP call, no headless
browser. The menu is a native `NSMenu` with `NSAttributedString`-rendered
colored percentages and SF Symbol status icons.
