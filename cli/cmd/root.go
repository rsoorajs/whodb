/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/tui"
	"github.com/clidey/whodb/cli/pkg/analytics"
	"github.com/clidey/whodb/cli/pkg/styles"
	"github.com/clidey/whodb/cli/pkg/updatecheck"
	"github.com/clidey/whodb/cli/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "whodb-cli",
	Short: "WhoDB CLI - Interactive database management tool",
	Long: `WhoDB CLI is an interactive, production-ready command-line interface for navigating SQL and NoSQL databases.

Features:
  - Split-pane TUI layouts (Single, Explore, Query, Full) — Ctrl+L to cycle
  - 8 color themes (Default, Monokai, Dracula, Nord, etc.) — Ctrl+T to cycle
  - Multi-database support (PostgreSQL, MySQL, SQLite, MongoDB, Redis, ClickHouse, etc.)
  - SQL editor with context-aware autocomplete, formatting (Ctrl+F), multi-tab buffers
  - External editor support (Ctrl+O opens $EDITOR)
  - ER diagram visualization (Ctrl+K)
  - EXPLAIN query plan viewer (Ctrl+X)
  - Data import/export (CSV, Excel) — Ctrl+G for import wizard
  - AI chat with streaming responses (OpenAI, Anthropic, Ollama, LM Studio)
  - SSH tunnel support for remote databases
  - Docker container auto-detection
  - Query bookmarks (Ctrl+B), history (Ctrl+H), command log (Ctrl+D)
  - Nested WHERE builder with AND/OR grouping
  - Connection profiles (Ctrl+P) — bundle connection + theme + settings
  - Data quality audit with configurable thresholds (Ctrl+U)
  - Read-only mode (Ctrl+Y)
  - JSON cell viewer, fish-style history suggestions

Press ? in any view for keyboard shortcuts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := viper.GetString("profile")
		if profileName != "" {
			return runWithProfile(profileName)
		}
		// Start TUI directly
		m := tui.NewMainModel()
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running interactive mode: %w", err)
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("no-update-check") || version.Version == "dev" {
			return
		}
		if result := updatecheck.Check(version.Version); result != nil {
			fmt.Fprintf(os.Stderr, "\nA new version of whodb-cli is available: %s → https://github.com/clidey/whodb/releases/latest\n", result.LatestVersion)
		}
	},
}

func Execute() {
	defer analytics.Shutdown()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runWithProfile loads a named profile, applies its settings, and connects.
func runWithProfile(name string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	profile := cfg.GetProfile(name)
	if profile == nil {
		return fmt.Errorf("profile %q not found", name)
	}

	conn, err := cfg.GetConnection(profile.Connection)
	if err != nil {
		return fmt.Errorf("profile %q references missing connection %q", name, profile.Connection)
	}

	// Apply display/query settings from the profile
	if profile.Theme != "" {
		if t := styles.GetThemeByName(profile.Theme); t != nil {
			styles.SetTheme(t)
			cfg.SetThemeName(profile.Theme)
		}
	}
	if profile.PageSize > 0 {
		cfg.SetPageSize(profile.PageSize)
	}
	if profile.TimeoutSeconds > 0 {
		cfg.Query.TimeoutSeconds = profile.TimeoutSeconds
	}

	m := tui.NewMainModelWithProfile(conn, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running interactive mode: %w", err)
	}
	return nil
}

func init() {
	cobra.OnInitialize(initConfig, initColorMode, initAnalytics)

	// Disable Cobra's default completion command; we provide our own with install support
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.PersistentFlags().String("profile", "", "load a named connection profile")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug mode")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	rootCmd.PersistentFlags().Bool("no-analytics", false, "disable anonymous usage analytics")
	rootCmd.PersistentFlags().Bool("no-update-check", false, "disable update check notifications")

	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("no-analytics", rootCmd.PersistentFlags().Lookup("no-analytics"))
	viper.BindPFlag("no-update-check", rootCmd.PersistentFlags().Lookup("no-update-check"))
}

func initAnalytics() {
	// Skip analytics if disabled via flag or env
	if viper.GetBool("no-analytics") || os.Getenv("WHODB_CLI_ANALYTICS_DISABLED") == "true" {
		return
	}

	// Initialize analytics (errors are silently ignored - analytics should never block CLI)
	_ = analytics.Initialize(version.Version)

	// Track CLI startup with the command being run
	if len(os.Args) > 1 {
		analytics.TrackCLIStartup(context.Background(), os.Args[1])
	} else {
		analytics.TrackCLIStartup(context.Background(), "tui")
	}
}

func initColorMode() {
	if viper.GetBool("no-color") {
		styles.DisableColor()
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		configDir := fmt.Sprintf("%s/.whodb-cli", home)
		if err := os.MkdirAll(configDir, 0700); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
			os.Exit(1)
		}
		// Enforce strict permissions
		_ = os.Chmod(configDir, 0700)

		viper.AddConfigPath(configDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("WHODB_CLI")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("debug") {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}
