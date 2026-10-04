package main

import (
	"context"
	"fmt"
	"os"

	"github.com/M5Devs/Total-Connect/internal/config"
	"github.com/M5Devs/Total-Connect/internal/core"
)

func main() {
	cfgPath := config.GetConfigPath()
	engine, err := core.NewRcloneEngine(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage engine: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Print status and detected remotes
	fmt.Println("=== Total Connect CLI POC ===")
	fmt.Printf("Config path: %s\n", cfgPath)
	if config.ConfigExists() {
		fmt.Println("Config status: Found")
	} else {
		fmt.Println("Config status: Not found (using defaults/empty)")
	}

	remotes, err := engine.ListRemotes(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing remotes: %v\n", err)
	} else {
		fmt.Printf("Detected remotes (%d):\n", len(remotes))
		for _, remote := range remotes {
			fmt.Printf("  - %s:\n", remote)
		}
	}
	fmt.Println()

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Usage:")
		fmt.Println("  tc-cli list [path]    List entries at the specified path (local or remote:path)")
		return
	}

	cmd := args[0]
	switch cmd {
	case "list":
		targetPath := "."
		if len(args) > 1 {
			targetPath = args[1]
		}
		listPath(ctx, engine, targetPath)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		fmt.Println("Usage:")
		fmt.Println("  tc-cli list [path]    List entries at the specified path (local or remote:path)")
		os.Exit(1)
	}
}

func listPath(ctx context.Context, engine core.StorageEngine, targetPath string) {
	fmt.Printf("Listing path: %s\n", targetPath)
	items, err := engine.ListEntries(ctx, targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing path %q: %v\n", targetPath, err)
		return
	}

	if len(items) == 0 {
		fmt.Println("  (empty directory or no entries)")
		return
	}

	for _, item := range items {
		typeStr := "[FILE]"
		if item.IsDir {
			typeStr = "[DIR ]"
		}
		fmt.Printf("  %s %-20s %10d bytes  %s\n",
			typeStr,
			item.Name,
			item.Size,
			item.ModTime.Format("2006-01-02 15:04:05"),
		)
	}
}
