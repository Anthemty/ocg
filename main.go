//go:build darwin

package main

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa

#include <stddef.h>
#include <stdlib.h>

// Implemented in app_darwin.m. All UI work is marshalled to the main queue.
void runApp(void);
void setStatusIcon(const void* bytes, size_t len);
void setStatusTooltip(const char* tooltip);
void updatePanelState(const char* stateJSON);

// Go callbacks invoked from Obj-C are declared via //export below; cgo
// generates their prototypes in _cgo_export.h automatically. Do NOT redeclare
// them here — a manual extern with const clashes with the generated decl.
*/
import "C"

import (
	"fmt"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// Provider names.
const (
	providerOpenCode = "opencode"
	providerDeepSeek = "deepseek"
	providerMiniMax  = "minimax"
)

// providers lists all known providers in sidebar order.
var providers = []string{providerOpenCode, providerDeepSeek, providerMiniMax}

// providerLabels maps provider name to display label.
var providerLabels = map[string]string{
	providerOpenCode: "OpenCode Go",
	providerDeepSeek: "DeepSeek",
	providerMiniMax:  "MiniMax",
}

// refreshMu serialises explicit refresh requests so we don't stack fetches.
var refreshMu sync.Mutex

func main() {
	// NSApplication's run loop must live on the main thread. Under darwin cgo
	// the main goroutine is the main thread; lock it and hand control to AppKit.
	runtime.LockOSThread()
	C.runApp()
}

// ---------- background refresh ----------

// backgroundRefresh runs in its own goroutine, fetching all providers and
// pushing the resulting state to the UI. It loops on a 15-minute ticker.
func backgroundRefresh() {
	refreshOnce()
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		refreshOnce()
	}
}

func refreshOnce() {
	refreshMu.Lock()
	defer refreshMu.Unlock()
	fetchAllProviders()
	pushUIState()
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
		cacheMu.Lock()
		providerCache[res.name] = res.r
		cacheMu.Unlock()
	}
	cacheMu.Lock()
	lastUpdated = time.Now()
	cacheMu.Unlock()
}

// pushUIState recomputes icon/tooltip/state and forwards them to the UI.
// Called only from the refresh goroutine and from cgo callbacks (main queue).
func pushUIState() {
	cacheMu.RLock()
	maxCrit := 0
	for _, cached := range providerCache {
		if cached != nil && cached.Err == nil && cached.Criticality > maxCrit {
			maxCrit = cached.Criticality
		}
	}
	cacheMu.RUnlock()

	icon := usageIconBytes(maxCrit)
	cBytes := C.CBytes(icon)
	defer C.free(cBytes)
	C.setStatusIcon(cBytes, C.size_t(len(icon)))

	tooltip := C.CString(fmt.Sprintf("Usage Monitor — worst: %d%%", maxCrit))
	defer C.free(unsafe.Pointer(tooltip))
	C.setStatusTooltip(tooltip)

	state := buildPanelStateJSON()
	cState := C.CString(state)
	defer C.free(unsafe.Pointer(cState))
	C.updatePanelState(cState)
}

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

// switchProvider changes the active provider and re-pushes state.
func switchProvider(name string) {
	cfg := loadCfg()
	if cfg.ActiveProvider == name {
		return
	}
	cfg.ActiveProvider = name
	_ = saveConfig(cfg)
	pushUIState()
}

// ---------- cgo callbacks (invoked from Obj-C, main queue) ----------

//export goOnReady
func goOnReady() {
	// Push a neutral icon immediately so the status item isn't blank while the
	// first fetch is in flight.
	pushIconOnly()
	go backgroundRefresh()
}

// pushIconOnly updates just the status icon/tooltip from cache without
// touching the panel state. Used for the initial placeholder.
func pushIconOnly() {
	icon := neutralIconBytes()
	cBytes := C.CBytes(icon)
	defer C.free(cBytes)
	C.setStatusIcon(cBytes, C.size_t(len(icon)))
	tip := C.CString("Usage Monitor — loading")
	defer C.free(unsafe.Pointer(tip))
	C.setStatusTooltip(tip)
}

//export goProviderSelected
func goProviderSelected(providerID *C.char) {
	id := C.GoString(providerID)
	switchProvider(id)
}

//export goRefreshRequested
func goRefreshRequested() {
	go refreshOnce()
}

//export goSaveCredentials
func goSaveCredentials(provider, field, value *C.char) {
	p := C.GoString(provider)
	f := C.GoString(field)
	v := C.GoString(value)
	go func() {
		cfg := loadCfg()
		switch {
		case p == providerOpenCode && f == "workspace_id":
			cfg.OpenCode.WorkspaceID = v
		case p == providerOpenCode && f == "auth_cookie":
			if v != "" && !startsWith(v, "auth=") {
				v = "auth=" + v
			}
			cfg.OpenCode.AuthCookie = v
		case p == providerDeepSeek && f == "api_key":
			cfg.DeepSeek.APIKey = v
		case p == providerMiniMax && f == "api_key":
			cfg.Minimax.APIKey = v
		default:
			return
		}
		_ = saveConfig(cfg)
		refreshOnce()
	}()
}

//export goQuitRequested
func goQuitRequested() {
	// Termination is handled on the AppKit side; this is a no-op hook kept
	// for symmetry / future teardown (e.g. flush config).
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
