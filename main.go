package main

import (
	"fmt"
	"time"

	"github.com/getlantern/systray"
)

var (
	mTitle     *systray.MenuItem
	mUpdated   *systray.MenuItem
	mRolling   *systray.MenuItem
	mWeekly    *systray.MenuItem
	mMonthly   *systray.MenuItem
	mRefresh   *systray.MenuItem
	mSetCookie *systray.MenuItem
	mSetWS     *systray.MenuItem
	mQuit      *systray.MenuItem
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(neutralIconBytes())
	systray.SetTooltip("OpenCode Go Usage")

	mTitle = systray.AddMenuItem("OpenCode Go Usage", "")
	mTitle.Disable()
	mUpdated = systray.AddMenuItem("Loading…", "")
	mUpdated.Disable()
	systray.AddSeparator()

	mRolling = systray.AddMenuItem(meterLine("Rolling", nil), "")
	mRolling.Disable()
	mWeekly = systray.AddMenuItem(meterLine("Weekly", nil), "")
	mWeekly.Disable()
	mMonthly = systray.AddMenuItem(meterLine("Monthly", nil), "")
	mMonthly.Disable()
	systray.AddSeparator()

	mRefresh = systray.AddMenuItem("Refresh Now", "Fetch latest usage data")
	mSetCookie = systray.AddMenuItem("Set Cookie…", "Configure auth cookie")
	mSetWS = systray.AddMenuItem("Set Workspace ID…", "Configure workspace ID")
	systray.AddSeparator()
	mQuit = systray.AddMenuItem("Quit", "")

	go backgroundRefresh()
	go handleClicks()
}

func onExit() {}

// ---------- background refresh ----------

func backgroundRefresh() {
	doFetch()
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		doFetch()
	}
}

func doFetch() {
	cfg, err := loadConfig()
	if err != nil {
		mUpdated.SetTitle("⚠ Config error")
		systray.SetIcon(neutralIconBytes())
		systray.SetTooltip(fmt.Sprintf("Config error: %v", err))
		return
	}
	if cfg.AuthCookie == "" {
		mUpdated.SetTitle("Not configured — Set Cookie…")
		mRolling.SetTitle(meterLine("Rolling", nil))
		mWeekly.SetTitle(meterLine("Weekly", nil))
		mMonthly.SetTitle(meterLine("Monthly", nil))
		systray.SetIcon(neutralIconBytes())
		systray.SetTooltip("Not configured — set auth cookie")
		return
	}
	data, err := fetchUsage(cfg)
	if err != nil {
		mUpdated.SetTitle("⚠ Fetch failed")
		systray.SetIcon(neutralIconBytes())
		systray.SetTooltip(fmt.Sprintf("Error: %v", err))
		return
	}
	updateMenu(data)
}

func updateMenu(data *UsageData) {
	mUpdated.SetTitle("Updated " + time.Now().Format("15:04"))
	mRolling.SetTitle(meterLine("Rolling", &data.Rolling))
	mWeekly.SetTitle(meterLine("Weekly", &data.Weekly))
	mMonthly.SetTitle(meterLine("Monthly", &data.Monthly))

	// Coloured menu-bar icon reflects monthly usage level.
	systray.SetIcon(usageIconBytes(data.Monthly.Percent))
	systray.SetTooltip(fmt.Sprintf("OCG — Rolling %d%% · Weekly %d%% · Monthly %d%%",
		data.Rolling.Percent, data.Weekly.Percent, data.Monthly.Percent))
}

// meterLine: "Rolling    2%   2h 1m"  with a coloured status dot emoji.
func meterLine(label string, m *Meter) string {
	if m == nil {
		return fmt.Sprintf("%-8s —", label)
	}
	return fmt.Sprintf("%-8s %3d%%   %s   %s", label, m.Percent, statusDot(m.Percent), formatDuration(m.ResetInSec))
}

// statusDot returns a coloured circle emoji by usage level.
func statusDot(pct int) string {
	switch {
	case pct < 50:
		return "🟢"
	case pct < 85:
		return "🟡"
	default:
		return "🔴"
	}
}

// ---------- menu click handlers ----------

func handleClicks() {
	for {
		select {
		case <-mRefresh.ClickedCh:
			go doFetch()
		case <-mSetCookie.ClickedCh:
			if promptSetCookie() {
				go doFetch()
			}
		case <-mSetWS.ClickedCh:
			if promptSetWorkspace() {
				go doFetch()
			}
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}
