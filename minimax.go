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
	// Response structure (from actual API call):
	// {"base_resp": {"status_code":0, "status_msg":"success"},
	//  "model_remains": [{
	//    "model_name": "general",
	//    "current_interval_remaining_percent": 76,   ← remaining % (not used)
	//    "current_weekly_remaining_percent": 100,
	//    "current_interval_total_count": 0,           ← always 0, ignore
	//    "current_interval_usage_count": 0,           ← always 0, ignore
	//    "current_interval_status": 1,
	//    "current_weekly_status": 3,
	//    "start_time": 1782104400000,                 ← ms epoch
	//    "end_time": 1782122400000,
	//    "weekly_start_time": 1782086400000,
	//    "weekly_end_time": 1782691200000,
	//    "remains_time": 3522055,
	//    "weekly_remains_time": 572322055
	//  }, ...]}

	// model_remains is at root level (no "data" wrapper).
	remains := getSlice(raw, "model_remains", "modelRemains")
	if remains == nil {
		// Try data wrapper as fallback.
		if dataObj := getMap(raw, "data"); dataObj != nil {
			remains = getSlice(dataObj, "model_remains", "modelRemains")
		}
	}
	if remains == nil {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("no model_remains")}
	}

	// Find the "general" model entry (skip video/image/etc).
	var entry map[string]any
	for _, item := range remains {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := getString(m, "model_name", "modelName")
		if name == "general" || name == "chat" || name == "text" {
			entry = m
			break
		}
	}
	if entry == nil {
		if m, ok := remains[0].(map[string]any); ok {
			entry = m
		}
	}
	if entry == nil {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("no model data")}
	}

	// The real data is in *_remaining_percent fields (remaining, not used).
	intervalRemainingPct := getFloat(entry,
		"current_interval_remaining_percent", "currentIntervalRemainingPercent")
	weeklyRemainingPct := getFloat(entry,
		"current_weekly_remaining_percent", "currentWeeklyRemainingPercent")

	intervalUsedPct := int(100 - intervalRemainingPct)
	weeklyUsedPct := int(100 - weeklyRemainingPct)

	// Compute reset times from ms-epoch timestamps.
	reset5h := ""
	if endMs := getFloat(entry, "end_time", "endTime"); endMs > 0 {
		t := time.Unix(int64(endMs/1000), 0)
		if d := time.Until(t); d > 0 {
			reset5h = fmt.Sprintf("%dh", int(d.Hours())+1)
		}
	}
	resetWeek := ""
	if weekEndMs := getFloat(entry, "weekly_end_time", "weeklyEndTime"); weekEndMs > 0 {
		t := time.Unix(int64(weekEndMs/1000), 0)
		if d := time.Until(t); d > 0 {
			resetWeek = fmt.Sprintf("%dd", int(d.Hours()/24)+1)
		}
	}

	criticality := intervalUsedPct
	if weeklyUsedPct > criticality {
		criticality = weeklyUsedPct
	}

	lines := []string{
		fmt.Sprintf("5h     %3d%% used  %s  %s", intervalUsedPct, statusDot(intervalUsedPct), reset5h),
		fmt.Sprintf("weekly %3d%% used  %s  %s", weeklyUsedPct, statusDot(weeklyUsedPct), resetWeek),
	}
	meters := []UsageMeter{
		{Label: "5h", Percent: intervalUsedPct, Detail: reset5h},
		{Label: "Weekly", Percent: weeklyUsedPct, Detail: resetWeek},
	}

	return &ProviderFetchResult{
		Criticality: criticality,
		Lines:       lines,
		Meters:      meters,
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
