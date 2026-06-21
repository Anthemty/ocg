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

func fetchUsage(cfg *Config) (*UsageData, error) {
	ws := cfg.WorkspaceID
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
				return fmt.Errorf("redirected to %s — session expired or invalid cookie", req.URL.String())
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ocg/1.0")
	req.Header.Set("Cookie", cfg.AuthCookie)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	matches := usageRe.FindAllSubmatch(body, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("could not find usage data in page — page structure may have changed")
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

	return &UsageData{
		Rolling:   usage["rollingUsage"],
		Weekly:    usage["weeklyUsage"],
		Monthly:   usage["monthlyUsage"],
		Plan:      "Go",
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}
