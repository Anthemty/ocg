package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// dialogInput shows a macOS native input dialog.
func dialogInput(title, prompt, defaultValue string) (string, bool) {
	title = strings.ReplaceAll(title, `"`, `\"`)
	prompt = strings.ReplaceAll(prompt, `"`, `\"`)
	defaultValue = strings.ReplaceAll(defaultValue, `"`, `\"`)
	script := fmt.Sprintf(
		`set result to display dialog "%s" with title "%s" default answer "%s"`,
		prompt, title, defaultValue,
	)
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", false
	}
	for _, part := range strings.Split(strings.TrimSpace(string(out)), ",") {
		part = strings.TrimSpace(part)
		if after, ok := strings.CutPrefix(part, "text returned:"); ok {
			return strings.TrimSpace(after), true
		}
	}
	return "", false
}

// dialogAlert shows a macOS native alert.
func dialogAlert(title, message string) {
	title = strings.ReplaceAll(title, `"`, `\"`)
	message = strings.ReplaceAll(message, `"`, `\"`)
	exec.Command("osascript", "-e",
		fmt.Sprintf(`display dialog "%s" with title "%s" buttons {"OK"} default button "OK"`,
			message, title)).Run()
}

// ---------- Configure dialogs ----------

func promptSetOpenCodeCookie() bool {
	val, ok := dialogInput("OpenCode Go", "Paste auth cookie value:", "")
	if !ok || val == "" {
		return false
	}
	if !strings.HasPrefix(val, "auth=") {
		val = "auth=" + val
	}
	cfg, err := loadConfig()
	if err != nil {
		cfg = &Config{ActiveProvider: "opencode"}
	}
	cfg.OpenCode.AuthCookie = val
	if err := saveConfig(cfg); err != nil {
		dialogAlert("Error", fmt.Sprintf("Failed to save config: %v", err))
		return false
	}
	return true
}

func promptSetOpenCodeWorkspace() bool {
	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{ActiveProvider: "opencode"}
	}
	val, ok := dialogInput("OpenCode Go", "Enter workspace ID:", cfg.OpenCode.WorkspaceID)
	if !ok || val == "" {
		return false
	}
	cfg.OpenCode.WorkspaceID = val
	if err := saveConfig(cfg); err != nil {
		dialogAlert("Error", fmt.Sprintf("Failed to save config: %v", err))
		return false
	}
	return true
}

func promptSetDeepSeekKey() bool {
	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{ActiveProvider: "deepseek"}
	}
	val, ok := dialogInput("DeepSeek", "Paste DeepSeek API key:", cfg.DeepSeek.APIKey)
	if !ok || val == "" {
		return false
	}
	cfg.DeepSeek.APIKey = val
	if err := saveConfig(cfg); err != nil {
		dialogAlert("Error", fmt.Sprintf("Failed to save config: %v", err))
		return false
	}
	return true
}

func promptSetMiniMaxKey() bool {
	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{ActiveProvider: "minimax"}
	}
	val, ok := dialogInput("MiniMax", "Paste MiniMax API key:", cfg.Minimax.APIKey)
	if !ok || val == "" {
		return false
	}
	cfg.Minimax.APIKey = val
	if err := saveConfig(cfg); err != nil {
		dialogAlert("Error", fmt.Sprintf("Failed to save config: %v", err))
		return false
	}
	return true
}

