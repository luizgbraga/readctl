package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/luizgbraga/readctl/internal/config"
	"github.com/luizgbraga/readctl/internal/storage"
	"github.com/luizgbraga/readctl/internal/tui"
)

func main() {
	// Handle subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "config":
			runConfig()
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}

	runApp()
}

func printHelp() {
	fmt.Println("readctl - Terminal-based reading companion")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  readctl          Launch the TUI application")
	fmt.Println("  readctl config   Configure API key and settings")
	fmt.Println("  readctl help     Show this help message")
}

func runConfig() {
	configPath, err := config.Path()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Try to load existing config
	cfg, _ := config.Load()
	if cfg == nil {
		cfg = &config.Config{
			Model: "claude-sonnet-4-5-20250929",
		}
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("readctl configuration")
	fmt.Println("─────────────────────")
	fmt.Printf("Config file: %s\n\n", configPath)

	// API Key
	currentKey := "(not set)"
	if cfg.AnthropicAPIKey != "" {
		// Mask the key, showing only last 4 chars
		currentKey = "****" + cfg.AnthropicAPIKey[len(cfg.AnthropicAPIKey)-4:]
	}
	fmt.Printf("Anthropic API Key [%s]: ", currentKey)
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey != "" {
		cfg.AnthropicAPIKey = apiKey
	}

	// Model
	fmt.Printf("Model [%s]: ", cfg.Model)
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)
	if model != "" {
		cfg.Model = model
	}

	// Firecrawl API Key (optional)
	currentFirecrawl := "(not set)"
	if cfg.FirecrawlAPIKey != "" {
		currentFirecrawl = "****" + cfg.FirecrawlAPIKey[len(cfg.FirecrawlAPIKey)-4:]
	}
	fmt.Printf("Firecrawl API Key (optional, enables web search) [%s]: ", currentFirecrawl)
	firecrawlKey, _ := reader.ReadString('\n')
	firecrawlKey = strings.TrimSpace(firecrawlKey)
	if firecrawlKey != "" {
		cfg.FirecrawlAPIKey = firecrawlKey
	}

	// Save
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✓ Configuration saved!")
}

func runApp() {
	// Set up debug logging if DEBUG env var is set
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set up debug logging: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	// Initialize storage
	db, err := storage.InitDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize storage: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please check that the data directory is writable.\n")
		os.Exit(1)
	}
	defer db.Close()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'readctl config' to set up your configuration.\n")
		os.Exit(1)
	}

	// Create and run Bubbletea program
	// In v2, alt screen is controlled via the View struct, not WithAltScreen()
	model := tui.New(db, cfg)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
