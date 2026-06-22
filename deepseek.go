package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// deepseekBalanceResp matches the GET /user/balance response.
type deepseekBalanceResp struct {
	IsAvailable  bool                  `json:"is_available"`
	BalanceInfos []deepseekBalanceInfo `json:"balance_infos"`
}

type deepseekBalanceInfo struct {
	Currency        string `json:"currency"`
	TotalBalance    string `json:"total_balance"`
	GrantedBalance  string `json:"granted_balance"`
	ToppedUpBalance string `json:"topped_up_balance"`
}

// fetchDeepSeek fetches balance from api.deepseek.com.
func fetchDeepSeek(cfg *Config) *ProviderFetchResult {
	key := cfg.DeepSeek.APIKey
	if key == "" {
		return &ProviderFetchResult{
			Criticality: 0,
			Err:         fmt.Errorf("not configured"),
			Lines:       []string{"— set API key to configure"},
		}
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", "https://api.deepseek.com/user/balance", nil)
	if err != nil {
		return &ProviderFetchResult{Criticality: 0, Err: err}
	}
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := client.Do(req)
	if err != nil {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("request failed: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))}
	}

	var bal deepseekBalanceResp
	if err := json.NewDecoder(resp.Body).Decode(&bal); err != nil {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("parse: %w", err)}
	}
	if !bal.IsAvailable {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("account is not available")}
	}
	if len(bal.BalanceInfos) == 0 {
		return &ProviderFetchResult{Criticality: 0, Err: fmt.Errorf("no balance info")}
	}

	info := bal.BalanceInfos[0]
	totalStr := info.TotalBalance
	grantedStr := info.GrantedBalance
	toppedUpStr := info.ToppedUpBalance

	// Parse numeric to compute criticality (¥0 → 100% critical).
	total := parseFloat(totalStr)

	// Lines: show readable amounts.
	cur := ""
	switch info.Currency {
	case "CNY":
		cur = "¥"
	case "USD":
		cur = "$"
	default:
		cur = info.Currency + " "
	}

	var criticality int
	if total <= 0 {
		criticality = 100
	} else {
		// Treat < ¥5 as yellow, ¥0 as red; scale linearly.
		if total >= 50 {
			criticality = 0
		} else {
			criticality = int((1 - total/50.0) * 100)
			if criticality < 0 {
				criticality = 0
			}
		}
	}

	lines := []string{
		fmt.Sprintf("Balance  %s%s  %s", cur, totalStr, statusDot(100-criticality)),
		fmt.Sprintf("Granted  %s%s", cur, grantedStr),
		fmt.Sprintf("Topped   %s%s", cur, toppedUpStr),
	}
	meters := []UsageMeter{
		{Label: "Balance", Percent: criticality, Detail: fmt.Sprintf("%s%s left", cur, totalStr)},
		{Label: "Granted", Percent: 0, Detail: cur + grantedStr},
		{Label: "Topped", Percent: 0, Detail: cur + toppedUpStr},
	}

	return &ProviderFetchResult{
		Criticality: criticality,
		Lines:       lines,
		Meters:      meters,
	}
}

func parseFloat(s string) float64 {
	var v float64
	_ = json.Unmarshal([]byte(s), &v)
	return v
}
