# ocg — Usage Monitor (macOS Menu Bar)

Multi-provider usage monitor for the macOS menu bar. Currently supports **OpenCode Go**, **DeepSeek** (balance), and **MiniMax** (token plan quota). Click the menu bar icon to open a native popover: a left sidebar shows each provider's brand logo, the right pane shows that provider's usage as progress bars. Switch providers by clicking a sidebar icon; configure credentials inline from the ⚙ Settings view.

The menu bar icon is a monochrome template gauge — a ring that fills proportionally to the worst usage across all providers, tinted automatically to match light/dark mode.

## Quick Start

```bash
make app                          # build → OCGTool.app
cp -R OCGTool.app ~/OCGTool.app   # copy out of source tree
open ~/OCGTool.app                # launch
```

## Popover

```
              ┌──┬──────────────────────┐
              │  │  OpenCode Go      ⚙  │
              │</>│  Updated 14:32:05    │
              │  │  Rolling    2%        │
[◔ gauge] ──▶│  │  ▓░░░░░░░░░░  2h 1m  │
              │🐳│  Weekly   24%        │
              │  │  ▓▓▓░░░░░░░░  18h    │
              │〰 │  Monthly  63%        │
              │  │  ▓▓▓▓▓▓░░░░  6d 11h  │
              │  │                       │
              │  │  Refresh       Quit   │
              └──┴───────────────────────┘
```

Click a sidebar icon to switch providers. Click **⚙** in the header to edit the active provider's credentials inline (auth cookie / API key) and **Save & Refresh**. The popover closes when you click outside it.

### OpenCode Go
- **Auth**: cookie-based (browser DevTools → Application → Cookies)
- **Data**: 3 time windows (rolling / weekly / monthly) with percentage used

### DeepSeek
- **Auth**: API key (Bearer token)
- **Data**: monetary balance (¥), granted vs topped-up

### MiniMax
- **Auth**: Token Plan API key
- **Data**: 5-hour window + weekly token quota usage

## Setup

Open the popover, click **⚙** in the header, edit the fields for the active provider, and click **Save & Refresh**.

Config stored at `~/.config/ocg/config.json`. Old single-provider config files are automatically migrated.

## Build

```bash
make        # builds OCGTool.app
make run    # builds and opens the app
make build  # plain binary
```

Requires Go 1.22+ and Xcode Command Line Tools (CGO for the native AppKit shell).

---

**Version 0.0.2** — native NSPopover shell (replaces systray), monochrome template gauge icon, brand-logo sidebar, inline credential editing.
