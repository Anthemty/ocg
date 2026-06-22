package main

import (
	"fmt"
	"time"

	"github.com/getlantern/systray"
)

// Provider names.
const (
	providerOpenCode = "opencode"
	providerDeepSeek = "deepseek"
	providerMiniMax  = "minimax"
)

// providers lists all known providers in menu order.
var providers = []string{providerOpenCode, providerDeepSeek, providerMiniMax}

// providerLabels maps provider name to display label.
var providerLabels = map[string]string{
	providerOpenCode: "OpenCode Go",
	providerDeepSeek: "DeepSeek",
	providerMiniMax:  "MiniMax",
}

// Maximum number of data rows any provider may return.
const maxDataRows = 3

var (
	mTitle      *systray.MenuItem
	mUpdated    *systray.MenuItem

	// Radio buttons — one per provider.
	mProviderRadio map[string]*systray.MenuItem

	// Data rows — maxDataRows disabled items.
	mDataRow [maxDataRows]*systray.MenuItem

	mRefresh            *systray.MenuItem
	mConfigureOpenCode  *systray.MenuItem
	mConfigureDS        *systray.MenuItem
	mConfigureMX        *systray.MenuItem
	mQuit               *systray.MenuItem
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	cfg := loadCfg()
	mProviderRadio = make(map[string]*systray.MenuItem)

	systray.SetIcon(neutralIconBytes())
	systray.SetTooltip("Usage Monitor")

	mTitle = systray.AddMenuItem("Usage Monitor", "")
	mTitle.Disable()
	mUpdated = systray.AddMenuItem("Loading…", "")
	mUpdated.Disable()
	systray.AddSeparator()

	// Radio group — one per provider.
	for _, p := range providers {
		label := providerLabels[p]
		checked := p == cfg.ActiveProvider
		item := systray.AddMenuItemCheckbox(label, "Show "+label+" usage", checked)
		mProviderRadio[p] = item
	}
	systray.AddSeparator()

	// Data rows.
	for i := range maxDataRows {
		mDataRow[i] = systray.AddMenuItem("", "")
		mDataRow[i].Disable()
	}

	systray.AddSeparator()

	mRefresh = systray.AddMenuItem("Refresh Now", "")
	systray.AddSeparator()
	mConfigureOpenCode = systray.AddMenuItem("OpenCode Cookie…", "Set OpenCode auth cookie")
	mConfigureDS = systray.AddMenuItem("DeepSeek Key…", "Set DeepSeek API key")
	mConfigureMX = systray.AddMenuItem("MiniMax Key…", "Set MiniMax API key")
	systray.AddSeparator()
	mQuit = systray.AddMenuItem("Quit", "")
	go backgroundRefresh()
	go handleClicks()
}

func onExit() {}

// loadCfg loads config with a safe default.
func loadCfg() *Config {
	cfg, err := loadConfig()
	if err != nil {
		return &Config{ActiveProvider: providerOpenCode}
	}
	if cfg.ActiveProvider == "" {
		cfg.ActiveProvider = providerOpenCode
	}
	return cfg
}

// ---------- background refresh ----------

func backgroundRefresh() {
	fetchAllProviders()
	updateMenu()

	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		fetchAllProviders()
		updateMenu()
	}
}

// fetchAllProviders fetches all providers in parallel.
func fetchAllProviders() {
	type result struct {
		name string
		r    *ProviderFetchResult
	}
	ch := make(chan result, len(providers))

	for _, p := range providers {
		p := p
		go func() {
			var r *ProviderFetchResult
			switch p {
			case providerOpenCode:
				r = fetchOpenCode(loadCfg())
			case providerDeepSeek:
				r = fetchDeepSeek(loadCfg())
			case providerMiniMax:
				r = fetchMiniMax(loadCfg())
			default:
				r = &ProviderFetchResult{Err: fmt.Errorf("unknown provider: %s", p)}
			}
			ch <- result{p, r}
		}()
	}

	for range providers {
		res := <-ch
		providerCache[res.name] = res.r
	}
}

func updateMenu() {
	cfg := loadCfg()
	active := cfg.ActiveProvider

	// Update radio checkboxes.
	for _, p := range providers {
		item := mProviderRadio[p]
		if p == active {
			item.Check()
		} else {
			item.Uncheck()
		}
	}

	// Update data rows from cache.
	r := providerCache[active]
	mUpdated.SetTitle("Updated " + time.Now().Format("15:04"))

	if r == nil || r.Err != nil {
		errMsg := "—"
		if r != nil && r.Err != nil {
			errMsg = r.Err.Error()
			if len(errMsg) > 45 {
				errMsg = errMsg[:45] + "…"
			}
		}
		mDataRow[0].SetTitle(errMsg)
		for i := 1; i < maxDataRows; i++ {
			mDataRow[i].SetTitle("")
		}
		systray.SetIcon(neutralIconBytes())
		return
	}

	// Set data lines.
	for i := 0; i < maxDataRows; i++ {
		if i < len(r.Lines) {
			mDataRow[i].SetTitle(r.Lines[i])
		} else {
			mDataRow[i].SetTitle("")
		}
	}

	// Icon: use the highest criticality across all providers.
	maxCrit := 0
	for _, cached := range providerCache {
		if cached != nil && cached.Err == nil && cached.Criticality > maxCrit {
			maxCrit = cached.Criticality
		}
	}
	systray.SetIcon(usageIconBytes(maxCrit))
	systray.SetTooltip(fmt.Sprintf("Usage Monitor — worst: %d%%", maxCrit))
}

// ---------- menu click handlers ----------

func handleClicks() {
	for {
		select {
		case <-mRefresh.ClickedCh:
			go func() {
				fetchAllProviders()
				updateMenu()
			}()
		case <-mConfigureOpenCode.ClickedCh:
			promptSetOpenCodeCookie()
			promptSetOpenCodeWorkspace()
			go func() {
				fetchAllProviders()
				updateMenu()
			}()
		case <-mConfigureDS.ClickedCh:
			promptSetDeepSeekKey()
			go func() {
				fetchAllProviders()
				updateMenu()
			}()
		case <-mConfigureMX.ClickedCh:
			promptSetMiniMaxKey()
			go func() {
				fetchAllProviders()
				updateMenu()
			}()
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		case <-mProviderRadio[providerOpenCode].ClickedCh:
			switchProvider(providerOpenCode)
		case <-mProviderRadio[providerDeepSeek].ClickedCh:
			switchProvider(providerDeepSeek)
		case <-mProviderRadio[providerMiniMax].ClickedCh:
			switchProvider(providerMiniMax)
		}
	}
}

func switchProvider(name string) {
	cfg := loadCfg()
	if cfg.ActiveProvider == name {
		return
	}
	cfg.ActiveProvider = name
	_ = saveConfig(cfg)

	// Update radio checks and data display.
	for _, p := range providers {
		if p == name {
			mProviderRadio[p].Check()
		} else {
			mProviderRadio[p].Uncheck()
		}
	}

	// Fill data rows for the newly selected provider.
	r := providerCache[name]
	mUpdated.SetTitle("Updated " + time.Now().Format("15:04"))
	if r != nil && r.Err == nil {
		for i := 0; i < maxDataRows; i++ {
			if i < len(r.Lines) {
				mDataRow[i].SetTitle(r.Lines[i])
			} else {
				mDataRow[i].SetTitle("")
			}
		}
	} else {
		errMsg := "—"
		if r != nil && r.Err != nil {
			errMsg = r.Err.Error()
			if len(errMsg) > 45 {
				errMsg = errMsg[:45] + "…"
			}
		}
		mDataRow[0].SetTitle(errMsg)
		for i := 1; i < maxDataRows; i++ {
			mDataRow[i].SetTitle("")
		}
	}
}
