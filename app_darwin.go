//go:build darwin

package main

import "encoding/json"

// panelState is the JSON contract pushed to the Obj-C UI.
//
// Field names are the wire format consumed by app_darwin.m; keep them stable.
type panelState struct {
	Active      string                       `json:"active"`
	UpdatedAt   string                       `json:"updated_at"`
	Worst       int                          `json:"worst"`
	Providers   []panelProvider              `json:"providers"`
	Results     map[string]panelResult       `json:"results"`
	Credentials map[string]panelCredentials  `json:"credentials"`
}

type panelProvider struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type panelResult struct {
	Criticality int          `json:"criticality"`
	Error       string       `json:"error,omitempty"`
	Meters      []UsageMeter `json:"meters,omitempty"`
}

// panelCredentials carries the editable fields for the settings form. It is
// intentionally a flat map per provider so the Obj-C side can look up fields
// without knowing the Go config layout.
type panelCredentials map[string]string

func buildPanelStateJSON() string {
	cfg := loadCfg()

	state := panelState{
		Active:      cfg.ActiveProvider,
		Providers:   make([]panelProvider, 0, len(providers)),
		Results:     make(map[string]panelResult, len(providers)),
		Credentials: make(map[string]panelCredentials, len(providers)),
	}
	for _, id := range providers {
		state.Providers = append(state.Providers, panelProvider{
			ID:    id,
			Label: providerLabels[id],
		})
	}

	cacheMu.RLock()
	if !lastUpdated.IsZero() {
		state.UpdatedAt = lastUpdated.Format("15:04:05")
	}
	for _, id := range providers {
		cached := providerCache[id]
		r := panelResult{}
		if cached == nil {
			r.Error = "not fetched"
		} else {
			r.Criticality = cached.Criticality
			if cached.Err != nil {
				r.Error = cached.Err.Error()
			} else {
				r.Meters = cached.Meters
			}
		}
		state.Results[id] = r
	}
	cacheMu.RUnlock()

	// Worst across all providers drives the icon, shown in the header too.
	maxCrit := 0
	cacheMu.RLock()
	for _, cached := range providerCache {
		if cached != nil && cached.Err == nil && cached.Criticality > maxCrit {
			maxCrit = cached.Criticality
		}
	}
	cacheMu.RUnlock()
	state.Worst = maxCrit

	// Credentials for the settings form (prefill). Read-only snapshot.
	oc := cfg.OpenCode
	ds := cfg.DeepSeek
	mx := cfg.Minimax
	state.Credentials[providerOpenCode] = panelCredentials{
		"workspace_id": oc.WorkspaceID,
		"auth_cookie":  oc.AuthCookie,
	}
	state.Credentials[providerDeepSeek] = panelCredentials{
		"api_key": ds.APIKey,
	}
	state.Credentials[providerMiniMax] = panelCredentials{
		"api_key": mx.APIKey,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return `{"active":"","providers":[],"results":{},"credentials":{}}`
	}
	return string(data)
}
