package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	jsonFlag := flag.Bool("json", false, "output as JSON")
	workspaceFlag := flag.String("workspace", "", "workspace ID (overrides config)")
	flag.Parse()

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	args := flag.Args()
	cmd := "usage"
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "login", "auth":
		if err := runLogin(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "usage", "check":
		if *workspaceFlag != "" {
			cfg.WorkspaceID = *workspaceFlag
		}
		if err := runUsage(cfg, *jsonFlag); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "cookie":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: ocg cookie <cookie-value>")
			os.Exit(1)
		}
		cfg.AuthCookie = args[1]
		if !strings.HasPrefix(cfg.AuthCookie, "auth=") {
			cfg.AuthCookie = "auth=" + cfg.AuthCookie
		}
		if err := saveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Cookie saved")

	case "workspace":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: ocg workspace <workspace-id>")
			os.Exit(1)
		}
		cfg.WorkspaceID = args[1]
		if err := saveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Workspace ID saved")

	case "config":
		fmt.Printf("Workspace ID: %s\n", cfg.WorkspaceID)
		if cfg.AuthCookie != "" {
			fmt.Printf("Auth Cookie: %s... (set)\n", safePrefix(cfg.AuthCookie, 30))
		} else {
			fmt.Println("Auth Cookie: (not set)")
		}
		cfgPath, _ := configPath()
		fmt.Printf("Config file: %s\n", cfgPath)

	default:
		if *workspaceFlag != "" {
			cfg.WorkspaceID = *workspaceFlag
		}
		if err := runUsage(cfg, *jsonFlag); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}

func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
