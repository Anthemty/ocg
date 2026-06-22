# ocg — Usage Monitor (macOS Menu Bar)

Multi-provider usage monitor for the macOS menu bar. Currently supports **OpenCode Go**, **DeepSeek** (balance), and **MiniMax** (token plan quota). Switch between providers via radio buttons in the dropdown menu.

## Quick Start

```bash
make app                          # build → OCGTool.app
cp -R OCGTool.app ~/OCGTool.app   # copy out of source tree
open ~/OCGTool.app                # launch
```

## Menu

```
[● colour-coded icon]              ← worst usage across all providers
  Usage Monitor
  Updated 14:32
  ──────────────
  ◎ OpenCode Go                    ← selected provider
  ○ DeepSeek
  ○ MiniMax
  ──────────────
  Rolling   2%   🟢   2h 1m         ← current provider's data
  Weekly   24%   🟢   18h
  Monthly  63%   🟡   6d 11h
  ──────────────
  Refresh Now
  Configure Providers…
  ──────────────
  Quit
```

Click a radio button to switch providers. The menu bar icon reflects the **worst** usage across all providers.

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

Use **Configure Providers…** from the menu, then follow the prompts to set credentials for each provider.

Config stored at `~/.config/ocg/config.json`. Old single-provider config files are automatically migrated.

## Build

```bash
make        # builds OCGTool.app
make run    # builds and opens the app
make build  # plain binary
```

Requires Go 1.22+ and Xcode Command Line Tools (CGO for systray).
