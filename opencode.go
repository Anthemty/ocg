package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const defaultWorkspace = "wrk_01KJHHDX0J71PAM2N7VDV6TKQF"

// Patterns for Solid.js embedded state: $R[N]={status:"ok",resetInSec:...,usagePercent:...}
var usageRe = regexp.MustCompile(`(rollingUsage|weeklyUsage|monthlyUsage):\$R\[\d+\]=\{status:"(\w+)",resetInSec:(\d+),usagePercent:(\d+)\}`)

// fetchOpenCode fetches and parses usage from opencode.ai.
// Returns a ProviderFetchResult. The result Lines hold up to 3 formatted rows.
func fetchOpenCode(cfg *Config) *ProviderFetchResult {
	oc := cfg.OpenCode
	if oc.AuthCookie == "" {
		return &ProviderFetchResult{
			Criticality: 0,
			Err:         fmt.Errorf("not configured"),
			Lines:       []string{"— set cookie to configure"},
		}
	}

	ws := oc.WorkspaceID
	if ws == "" {
		ws = defaultWorkspace
	}

	url := fmt.Sprintf("https://opencode.ai/workspace/%s/go", ws)
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects — session may be expired")
			}
			if len(via) > 0 && !strings.Contains(req.URL.String(), "opencode.ai/workspace") {
				return fmt.Errorf("redirected — session expired or invalid cookie")
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &ProviderFetchResult{Criticality: 0, Err: err}
	}
	req.Header.Set("User-Agent", "ocg/1.0")
	req.Header.Set("Cookie", oc.AuthCookie)

	resp, err := client.Do(req)
	if err != nil {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("request failed: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("read body: %w", err)}
	}

	matches := usageRe.FindAllSubmatch(body, -1)
	if len(matches) == 0 {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("could not find usage data in page")}
	}

	usage := make(map[string]Meter)
	for _, m := range matches {
		label := string(m[1])
		status := string(m[2])
		resetInSec, _ := strconv.Atoi(string(m[3]))
		percent, _ := strconv.Atoi(string(m[4]))
		usage[label] = Meter{
			Percent:    percent,
			ResetInSec: resetInSec,
			Status:     status,
		}
	}

	r := usage["rollingUsage"]
	w := usage["weeklyUsage"]
	mo := usage["monthlyUsage"]

	maxPct := max(r.Percent, w.Percent, mo.Percent)
	lines := []string{
		fmt.Sprintf("Rolling %3d%% %s %s", r.Percent, statusDot(r.Percent), formatDuration(r.ResetInSec)),
		fmt.Sprintf("Weekly  %3d%% %s %s", w.Percent, statusDot(w.Percent), formatDuration(w.ResetInSec)),
		fmt.Sprintf("Monthly %3d%% %s %s", mo.Percent, statusDot(mo.Percent), formatDuration(mo.ResetInSec)),
	}

	return &ProviderFetchResult{
		Criticality: maxPct,
		Lines:       lines,
	}
}
