package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// fetchMiniMax fetches token plan remains from minimax.
// The response JSON is notoriously inconsistent in field naming (snake_case
// vs camelCase). We parse into a raw map and try both conventions.
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

	// Try multiple hosts — official docs use www.minimax.io, alternatives exist.
	var raw map[string]any
	var errors []string
	for _, host := range []string{"https://www.minimax.io", "https://api.minimax.io"} {
		url := host + "/v1/token_plan/remains"
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", url, err))
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: read %v", url, err))
			continue
		}
		if resp.StatusCode != 200 {
			errors = append(errors, fmt.Sprintf("%s: HTTP %d %s", url, resp.StatusCode, strings.TrimSpace(string(body))))
			continue
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			errors = append(errors, fmt.Sprintf("%s: parse %v", url, err))
			continue
		}
		errors = nil // got a valid response
		break
	}
	if errors != nil {
		msg := strings.Join(errors, "; ")
		if len(msg) > 80 {
			msg = msg[:80] + "…"
		}
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("request failed: %s", msg)}
	}

	// Check base_resp / baseResp status.
	if code := getInt(raw, "status_code"); code != 0 {
		msg := getString(raw, "status_msg")
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("api error: %s", msg)}
	}

	// Try data wrapper first, then fall back to root level.
	dataObj := getMap(raw, "data")
	if dataObj == nil {
		dataObj = raw // flat response with model_remains at root
	}


	// Get model_remains array (try both naming conventions).
	remains := getSlice(dataObj, "model_remains", "modelRemains")
	if remains == nil {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("no model_remains, keys: %v", keysOf(dataObj))}
	}

	// Find first text model with non-zero total count (interval or weekly).
	var entry map[string]any
	for _, item := range remains {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		total := getFloat(m, "current_interval_total_count", "currentIntervalTotalCount")
		weekly := getFloat(m, "current_weekly_total_count", "currentWeeklyTotalCount")
		pct := getFloat(m, "usage_percent", "usagePercent")
		if total > 0 || weekly > 0 || pct > 0 {
			entry = m
			break
		}
	}
	if entry == nil {
		if len(remains) > 0 {
			if m, ok := remains[0].(map[string]any); ok {
				return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("first model entry keys: %v", keysOf(m))}
			}
		}
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("no model with quota data (remains len=%d)", len(remains))}
	}

	// Read fields: usage_count = remaining, total_count = total.
	total := getFloat(entry, "current_interval_total_count", "currentIntervalTotalCount")
	remaining := getFloat(entry, "current_interval_usage_count", "currentIntervalUsageCount")
	weeklyTotal := getFloat(entry, "current_weekly_total_count", "currentWeeklyTotalCount")
	weeklyRemaining := getFloat(entry, "current_weekly_usage_count", "currentWeeklyUsageCount")
	usagePct := getFloat(entry, "usage_percent", "usagePercent") // remaining %

	// Compute used, preferring count-based fields.
	var intervalUsedPct, weeklyUsedPct int

	if total > 0 && remaining >= 0 {
		used := total - remaining
		intervalUsedPct = int(used / total * 100)
	} else if usagePct > 0 {
		// usagePct is remaining percent per MiniMax docs.
		intervalUsedPct = int(100 - usagePct)
	}

	if weeklyTotal > 0 && weeklyRemaining >= 0 {
		used := weeklyTotal - weeklyRemaining
		weeklyUsedPct = int(used / weeklyTotal * 100)
	} else {
		weeklyUsedPct = intervalUsedPct // fallback
	}

	criticality := intervalUsedPct
	if weeklyUsedPct > criticality {
		criticality = weeklyUsedPct
	}

	lines := []string{
		fmt.Sprintf("5h window  %3d%% used  %s", intervalUsedPct, statusDot(intervalUsedPct)),
		fmt.Sprintf("weekly     %3d%% used  %s", weeklyUsedPct, statusDot(weeklyUsedPct)),
	}

	return &ProviderFetchResult{
		Criticality: criticality,
		Lines:       lines,
	}
}

// ---------- flexible JSON helpers ----------

func getInt(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch n := v.(type) {
			case float64:
				return int(n)
			case int:
				return n
			case int64:
				return int(n)
			}
		}
	}
	// Check nested base_resp/baseResp.
	for _, baseKey := range []string{"base_resp", "baseResp"} {
		if sub := getMap(m, baseKey); sub != nil {
			return getInt(sub, keys...)
		}
	}
	return 0
}

func getString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func getFloat(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch n := v.(type) {
			case float64:
				return n
			case int:
				return float64(n)
			case int64:
				return float64(n)
			case string:
				var f float64
				if json.Unmarshal([]byte(n), &f) == nil {
					return f
				}
			}
		}
	}
	return 0
}

func getMap(m map[string]any, keys ...string) map[string]any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if mv, ok := v.(map[string]any); ok {
				return mv
			}
		}
	}
	return nil
}

func getSlice(m map[string]any, keys ...string) []any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.([]any); ok {
				return s
			}
		}
	}
	return nil
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
