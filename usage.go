package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const defaultWorkspace = "wrk_01KJHHDX0J71PAM2N7VDV6TKQF"

// Patterns for Solid.js embedded state: $R[N]={status:"ok",resetInSec:...,usagePercent:...}
var usageRe = regexp.MustCompile(`(rollingUsage|weeklyUsage|monthlyUsage):\$R\[\d+\]=\{status:"(\w+)",resetInSec:(\d+),usagePercent:(\d+)\}`)

func runUsage(cfg *Config, jsonOut bool) error {
	ws := cfg.WorkspaceID
	if ws == "" {
		ws = defaultWorkspace
	}
	cookie := cfg.AuthCookie
	if cookie == "" {
		return fmt.Errorf("no auth cookie set — run 'ocg login' first")
	}

	url := fmt.Sprintf("https://opencode.ai/workspace/%s/go", ws)

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects — session may be expired")
			}
			if len(via) > 0 && !strings.Contains(req.URL.String(), "opencode.ai/workspace") {
				return fmt.Errorf("redirected to %s — session expired or invalid cookie", req.URL.String())
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "ocg/1.0")
	req.Header.Set("Cookie", cookie)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	matches := usageRe.FindAllSubmatch(body, -1)
	if len(matches) == 0 {
		return fmt.Errorf("could not find usage data in page — page structure may have changed")
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

	u := UsageData{
		Rolling:   usage["rollingUsage"],
		Weekly:    usage["weeklyUsage"],
		Monthly:   usage["monthlyUsage"],
		Plan:      "Go",
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(u)
	}

	fmt.Println("OpenCode Go Usage")
	fmt.Printf("  Plan: %s\n\n", u.Plan)
	printMeter("Rolling", u.Rolling)
	printMeter("Weekly", u.Weekly)
	printMeter("Monthly", u.Monthly)
	return nil
}

func printMeter(label string, m Meter) {
	fmt.Printf("  %s: %3d%% used  (resets in %s)\n", label, m.Percent, formatDuration(m.ResetInSec))
	fmt.Printf("  %s %s  %d%%\n\n", label, bar(m.Percent, 45), m.Percent)
}
