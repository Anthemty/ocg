package main

import (
	"fmt"
	"time"

	"github.com/progrium/darwinkit/dispatch"
	"github.com/progrium/darwinkit/macos"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/progrium/darwinkit/objc"
)

// Menu item references held across the app lifetime.
var (
	statusItem appkit.StatusItem
	menu       appkit.Menu

	mTitle   appkit.MenuItem
	mUpdated appkit.MenuItem
	mRolling appkit.MenuItem
	mWeekly  appkit.MenuItem
	mMonthly appkit.MenuItem
	mRefresh appkit.MenuItem
	mCookie  appkit.MenuItem
	mWS      appkit.MenuItem
	mQuit    appkit.MenuItem
)

func main() {
	macos.RunApp(func(app appkit.Application, delegate *appkit.ApplicationDelegate) {
		app.SetActivationPolicy(appkit.ApplicationActivationPolicyAccessory)
		setupStatusItem(app)
		go backgroundRefresh()
	})
}

func setupStatusItem(app appkit.Application) {
	statusItem = appkit.StatusBar_SystemStatusBar().StatusItemWithLength(appkit.VariableStatusItemLength)
	objc.Retain(&statusItem)

	menu = appkit.NewMenuWithTitle("main")
	menu.SetAutoenablesItems(false)
	statusItem.SetMenu(menu)

	// Header
	mTitle = appkit.NewMenuItemWithTitleActionKeyEquivalent("OpenCode Go Usage", objc.Selector{}, "")
	mTitle.SetEnabled(false)
	mTitle.SetAttributedTitle(sectionHeader("OpenCode Go Usage"))
	menu.AddItem(mTitle)

	mUpdated = appkit.NewMenuItemWithTitleActionKeyEquivalent("Loading…", objc.Selector{}, "")
	mUpdated.SetEnabled(false)
	menu.AddItem(mUpdated)

	menu.AddItem(appkit.MenuItem_SeparatorItem())

	// Meters
	mRolling = appkit.NewMenuItemWithTitleActionKeyEquivalent("", objc.Selector{}, "")
	mRolling.SetEnabled(false)
	mRolling.SetAttributedTitle(meterLine("Rolling", nil))
	menu.AddItem(mRolling)

	mWeekly = appkit.NewMenuItemWithTitleActionKeyEquivalent("", objc.Selector{}, "")
	mWeekly.SetEnabled(false)
	mWeekly.SetAttributedTitle(meterLine("Weekly", nil))
	menu.AddItem(mWeekly)

	mMonthly = appkit.NewMenuItemWithTitleActionKeyEquivalent("", objc.Selector{}, "")
	mMonthly.SetEnabled(false)
	mMonthly.SetAttributedTitle(meterLine("Monthly", nil))
	menu.AddItem(mMonthly)

	menu.AddItem(appkit.MenuItem_SeparatorItem())

	mRefresh = appkit.NewMenuItemWithAction("Refresh Now", "", func(objc.Object) { go doFetch() })
	menu.AddItem(mRefresh)
	mCookie = appkit.NewMenuItemWithAction("Set Cookie…", "", func(objc.Object) {
		go func() {
			if promptSetCookie() {
				doFetch()
			}
		}()
	})
	menu.AddItem(mCookie)
	mWS = appkit.NewMenuItemWithAction("Set Workspace ID…", "", func(objc.Object) {
		go func() {
			if promptSetWorkspace() {
				doFetch()
			}
		}()
	})
	menu.AddItem(mWS)

	menu.AddItem(appkit.MenuItem_SeparatorItem())

	mQuit = appkit.NewMenuItemWithAction("Quit", "q", func(objc.Object) { app.Terminate(nil) })
	menu.AddItem(mQuit)

	setStatusIcon(0)
	statusItem.Button().SetToolTip("OpenCode Go Usage")
}


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
		dispatch.MainQueue().DispatchAsync(func() {
			mUpdated.SetAttributedTitle(muted("⚠ Config error"))
			statusItem.Button().SetToolTip(fmt.Sprintf("Config error: %v", err))
			setStatusIcon(0)
		})
		return
	}
	if cfg.AuthCookie == "" {
		dispatch.MainQueue().DispatchAsync(func() {
			mUpdated.SetAttributedTitle(muted("Not configured — set cookie"))
			mRolling.SetAttributedTitle(meterLine("Rolling", nil))
			mWeekly.SetAttributedTitle(meterLine("Weekly", nil))
			mMonthly.SetAttributedTitle(meterLine("Monthly", nil))
			statusItem.Button().SetToolTip("Not configured — set auth cookie")
			setStatusIcon(0)
		})
		return
	}
	data, err := fetchUsage(cfg)
	if err != nil {
		dispatch.MainQueue().DispatchAsync(func() {
			mUpdated.SetAttributedTitle(muted("⚠ Fetch failed"))
			statusItem.Button().SetToolTip(fmt.Sprintf("Error: %v", err))
			setStatusIcon(0)
		})
		return
	}
	dispatch.MainQueue().DispatchAsync(func() {
		mUpdated.SetAttributedTitle(muted("Updated " + time.Now().Format("15:04")))
		mRolling.SetAttributedTitle(meterLine("Rolling", &data.Rolling))
		mWeekly.SetAttributedTitle(meterLine("Weekly", &data.Weekly))
		mMonthly.SetAttributedTitle(meterLine("Monthly", &data.Monthly))
		statusItem.Button().SetToolTip(fmt.Sprintf("OCG — Rolling %d%% · Weekly %d%% · Monthly %d%%",
			data.Rolling.Percent, data.Weekly.Percent, data.Monthly.Percent))
		setStatusIcon(data.Monthly.Percent)
	})
}

// ---------- SF Symbol status icon ----------

// setStatusIcon swaps the menu bar symbol based on the monthly usage level.
func setStatusIcon(monthlyPct int) {
	name := "gauge.with.dots.needle.0percent"
	switch {
	case monthlyPct <= 0:
		name = "gauge.with.dots.needle.0percent"
	case monthlyPct < 50:
		name = "gauge.low"
	case monthlyPct < 85:
		name = "gauge.medium"
	default:
		name = "gauge.high"
	}
	img := appkit.Image_ImageWithSystemSymbolNameAccessibilityDescription(name, "OpenCode Go usage")
	img.SetTemplate(true)
	statusItem.Button().SetImage(img)
}

// ---------- attributed string builders ----------

const (
	attrFont  = "NSFont"
	attrColor = "NSColor"
)

var (
	menuFontMed = appkit.Font_SystemFontOfSizeWeight(0, appkit.FontWeightSemibold)
	smallFont   = appkit.Font_SystemFontOfSize(11)
)

// sectionHeader: bold title for the top of the menu.
func sectionHeader(s string) foundation.MutableAttributedString {
	a := foundation.NewMutableAttributedString()
	a.InitWithString(s)
	applyAttr(a, 0, uint64(len(s)), attrFont, menuFontMed)
	return a
}

func muted(s string) foundation.MutableAttributedString {
	a := foundation.NewMutableAttributedString()
	a.InitWithString(s)
	applyAttr(a, 0, uint64(len(s)), attrFont, smallFont)
	applyAttr(a, 0, uint64(len(s)), attrColor, appkit.Color_TertiaryLabelColor())
	return a
}

// meterLine: "Rolling    2%   resets in 2h 1m"
func meterLine(label string, m *Meter) foundation.MutableAttributedString {
	if m == nil {
		a := foundation.NewMutableAttributedString()
		a.InitWithString(fmt.Sprintf("%-9s  —", label))
		applyAttr(a, 0, uint64(len(label)), attrColor, appkit.Color_SecondaryLabelColor())
		return a
	}
	pctStr := fmt.Sprintf("%3d%%", m.Percent)
	resetStr := "resets in " + formatDuration(m.ResetInSec)
	full := fmt.Sprintf("%-9s %s   %s", label, pctStr, resetStr)

	a := foundation.NewMutableAttributedString()
	a.InitWithString(full)
	labelLen := uint64(len(label))
	pctStart := labelLen + 1
	pctLen := uint64(len(pctStr))
	resetStart := pctStart + pctLen + 3
	resetLen := uint64(len(resetStr))

	applyAttr(a, 0, labelLen, attrColor, appkit.Color_SecondaryLabelColor())
	applyAttr(a, pctStart, pctLen, attrColor, usageColor(m.Percent))
	applyAttr(a, pctStart, pctLen, attrFont, menuFontMed)
	applyAttr(a, resetStart, resetLen, attrColor, appkit.Color_TertiaryLabelColor())
	return a
}

// usageColor: green/yellow/red system color by percentage.
func usageColor(pct int) appkit.Color {
	switch {
	case pct < 50:
		return appkit.Color_SystemGreenColor()
	case pct < 85:
		return appkit.Color_SystemYellowColor()
	default:
		return appkit.Color_SystemRedColor()
	}
}

func applyAttr(a foundation.MutableAttributedString, loc, length uint64, key string, value objc.IObject) {
	r := foundation.Range{Location: loc, Length: length}
	a.AddAttributeValueRange(foundation.AttributedStringKey(key), value, r)
}
