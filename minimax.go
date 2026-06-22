package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// minimaxRemainsResp matches GET /v1/token_plan/remains.
type minimaxRemainsResp struct {
	BaseResp minimaxBaseResp      `json:"base_resp"`
	Data     minimaxData          `json:"data"`
}

type minimaxBaseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type minimaxData struct {
	ModelRemains []minimaxModelRemain `json:"model_remains"`
}

type minimaxModelRemain struct {
	ModelName                  string  `json:"model_name"`
	CurrentIntervalUsageCount  float64 `json:"current_interval_usage_count"`
	CurrentIntervalTotalCount  float64 `json:"current_interval_total_count"`
	CurrentWeeklyUsageCount    float64 `json:"current_weekly_usage_count"`
	CurrentWeeklyTotalCount    float64 `json:"current_weekly_total_count"`
	UsagePercent               float64 `json:"usage_percent"`
	StartTime                  string  `json:"start_time"`
	EndTime                    string  `json:"end_time"`
}

// fetchMiniMax fetches token plan remains from minimax.
func fetchMiniMax(cfg *Config) *ProviderFetchResult {
	key := cfg.Minimax.APIKey
	if key == "" {
		return &ProviderFetchResult{
			Criticality: 0,
			Err:         fmt.Errorf("not configured"),
			Lines:       []string{"— set API key to configure"},
		}
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", "https://api.minimax.io/v1/token_plan/remains", nil)
	if err != nil {
		return &ProviderFetchResult{Criticality: 0, Err: err}
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("request failed: %w", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("read body: %w", err)}
	}

	if resp.StatusCode != 200 {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))}
	}

	var result minimaxRemainsResp
	if err := json.Unmarshal(body, &result); err != nil {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("parse: %w", err)}
	}
	if result.BaseResp.StatusCode != 0 {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("api error: %s", result.BaseResp.StatusMsg)}
	}

	// Find the first text model entry (non-zero total).
	var entry *minimaxModelRemain
	for i := range result.Data.ModelRemains {
		m := &result.Data.ModelRemains[i]
		if m.CurrentIntervalTotalCount > 0 {
			entry = m
			break
		}
	}
	if entry == nil {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("no model with quota data")}
	}

	// MiniMax "usage_percent" means REMAINING. Convert to used.
	intervalUsed := entry.CurrentIntervalTotalCount - entry.CurrentIntervalUsageCount
	weeklyUsed := entry.CurrentWeeklyTotalCount - entry.CurrentWeeklyUsageCount

	intervalUsedPct := int(intervalUsed / entry.CurrentIntervalTotalCount * 100)
	weeklyUsedPct := int(weeklyUsed / entry.CurrentWeeklyTotalCount * 100)

	criticality := max(intervalUsedPct, weeklyUsedPct)

	// Format reset time from end_time if available.
	reset5h := ""
	resetWeek := ""
	if entry.EndTime != "" {
		t, err := time.Parse(time.RFC3339, entry.EndTime)
		if err == nil {
			reset5h = fmt.Sprintf("%dh window", int(time.Until(t).Hours())+1)
		}
	}

	lines := []string{
		fmt.Sprintf("5h window  %3d%% used  %s  %s", intervalUsedPct, statusDot(intervalUsedPct), reset5h),
		fmt.Sprintf("weekly     %3d%% used  %s  %s", weeklyUsedPct, statusDot(weeklyUsedPct), resetWeek),
	}

	return &ProviderFetchResult{
		Criticality: criticality,
		Lines:       lines,
	}
}
