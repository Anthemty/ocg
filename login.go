package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// dialogInput shows a macOS native input dialog. Returns the entered text and
// whether the user confirmed (OK) or cancelled.
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
		return "", false // user cancelled or dismissed
	}
	// output: "button returned:OK, text returned:value"
	for _, part := range strings.Split(strings.TrimSpace(string(out)), ",") {
		part = strings.TrimSpace(part)
		if after, ok := strings.CutPrefix(part, "text returned:"); ok {
			return strings.TrimSpace(after), true
		}
	}
	return "", false
}

// dialogAlert shows a macOS native alert with a single OK button.
func dialogAlert(title, message string) {
	title = strings.ReplaceAll(title, `"`, `\"`)
	message = strings.ReplaceAll(message, `"`, `\"`)
	exec.Command("osascript", "-e",
		fmt.Sprintf(`display dialog "%s" with title "%s" buttons {"OK"} default button "OK"`,
			message, title)).Run()
}

// promptSetCookie shows a dialog asking for the auth cookie and saves it.
func promptSetCookie() bool {
	val, ok := dialogInput("OCG Usage", "Paste auth cookie value:", "")
	if !ok || val == "" {
		return false
	}
	if !strings.HasPrefix(val, "auth=") {
		val = "auth=" + val
	}
	cfg, err := loadConfig()
	if err != nil {
		cfg = &Config{}
	}
	cfg.AuthCookie = val
	if err := saveConfig(cfg); err != nil {
		dialogAlert("Error", fmt.Sprintf("Failed to save config: %v", err))
		return false
	}
	return true
}

// promptSetWorkspace shows a dialog asking for the workspace ID and saves it.
func promptSetWorkspace() bool {
	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}
	val, ok := dialogInput("OCG Usage", "Enter workspace ID:", cfg.WorkspaceID)
	if !ok || val == "" {
		return false
	}
	cfg.WorkspaceID = val
	if err := saveConfig(cfg); err != nil {
		dialogAlert("Error", fmt.Sprintf("Failed to save config: %v", err))
		return false
	}
	return true
}
