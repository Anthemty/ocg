package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func runLogin(args []string) error {
	cfg, _ := loadConfig()
	if cfg == nil {
		cfg = &Config{}
	}

	ws := cfg.WorkspaceID
	if ws == "" {
		ws = defaultWorkspace
	}

	fmt.Println("OpenCode Go Usage — Login")
	fmt.Println()
	fmt.Printf("1. Open this URL in your browser:\n   https://opencode.ai/workspace/%s/go\n", ws)
	fmt.Println()
	fmt.Println("2. Log in with GitHub or Google if prompted")
	fmt.Println()
	fmt.Println("3. Open DevTools (F12) → Application → Cookies → opencode.ai")
	fmt.Println("   Copy the full 'auth' cookie value")
	fmt.Println()
	fmt.Println("4. Paste the cookie value below and press Enter:")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	cookie, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	cookie = strings.TrimSpace(cookie)
	if cookie == "" {
		return fmt.Errorf("no cookie provided")
	}

	// Normalize: wrap with "auth=" if just the value (not the full cookie header)
	if !strings.HasPrefix(cookie, "auth=") {
		cookie = "auth=" + cookie
	}

	cfg.AuthCookie = cookie

	fmt.Print("Workspace ID (press Enter for default): ")
	id, _ := reader.ReadString('\n')
	id = strings.TrimSpace(id)
	if id != "" {
		cfg.WorkspaceID = id
	} else if cfg.WorkspaceID == "" {
		cfg.WorkspaceID = defaultWorkspace
	}

	if err := saveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println("✓ Cookie saved to ~/.config/ocg/config.json")
	fmt.Println("  Run 'ocg' to check usage.")
	return nil
}
