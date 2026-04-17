/*
 * Copyright 2025 Clidey, Inc.
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
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/connectionopts"
	"github.com/clidey/whodb/cli/internal/docker"
	"github.com/clidey/whodb/cli/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var _ = tea.ProgramOption(nil) // Used in RunE

var (
	dbType               string
	host                 string
	port                 int
	username             string
	database             string
	schema               string
	name                 string
	passwordFromStdin    bool
	useDocker            bool
	connectSSLMode       string
	connectSSLCA         string
	connectSSLCert       string
	connectSSLKey        string
	connectSSLServerName string
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a database",
	Long: `Connect to a database and start the interactive TUI.

Usage modes:
  1) Flags path
     Provide --type and --database (optionally --host, --port, --user, --name).
     For databases that need a password, you'll be prompted on a TTY.
     For non-TTY (piped/CI), pass --password and pipe on stdin.
     If you pass --name, the connection is saved for later use.

  2) Docker auto-detection
     Use --docker to detect running database containers and connect.

  3) TUI connection form
     If required flags are missing, the interactive connection form opens.
     Docker containers appear automatically in the connection list.
`,
	Example: `
  # Open connection form (interactive — shows saved + Docker connections)
  whodb-cli connect

  # Connect to PostgreSQL
  whodb-cli connect --type postgres --host localhost --user alice --database app

  # Connect to SQLite (no password needed)
  whodb-cli connect --type sqlite3 --database ./app.db

  # Auto-detect Docker database containers
  whodb-cli connect --docker

  # Non-interactive: read password from stdin
  printf "%s\n" "$DB_PASS" | whodb-cli connect --type postgres --host localhost --user alice --database app --password
  whodb-cli connect --type sqlite --host ./app.db --database ./app.db --name app-sqlite

  # Connect with SSL
  whodb-cli connect --type postgres --host localhost --user alice --database app --ssl-mode verify-ca --ssl-ca ./ca.pem`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// --docker: detect running database containers and connect to the first match
		if useDocker {
			containers := docker.DetectContainers()
			if len(containers) == 0 {
				return fmt.Errorf("no running database containers detected (is Docker running?)")
			}
			c := containers[0]
			fmt.Fprintf(os.Stderr, "Detected %d container(s); connecting to %s (%s on port %d)\n", len(containers), c.Name, c.Type, c.Port)
			conn := config.Connection{
				Type:     c.Type,
				Host:     "localhost",
				Port:     c.Port,
				Database: database,
			}
			m := tui.NewMainModelWithConnection(&conn)
			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("error running interactive mode: %w", err)
			}
			return nil
		}

		resolvedType, typeKnown := lookupDatabaseType(dbType)
		if dbType != "" && !typeKnown {
			return fmt.Errorf("unsupported database type %q", dbType)
		}

		// If type and database are provided, connect directly.
		// Username is optional for file-based databases (SQLite, DuckDB) and
		// some NoSQL databases (Redis, MongoDB).
		if typeKnown && (database != "" || !resolvedType.RequiredFields.Database) {
			// Use defaults if not provided
			if host == "" {
				if isFileBasedDatabaseType(string(resolvedType.ID)) {
					// File-based databases use the database path as host
					host = database
				} else {
					host = "localhost"
				}
			}
			if port == 0 {
				port = getDefaultPort(dbType)
			} else if port < 1024 || port > 65535 {
				return fmt.Errorf("invalid port number %d: must be between 1024 and 65535 (ports below 1024 are system reserved)", port)
			}

			// Secure password prompt — skip for databases that don't need credentials
			var password string
			needsPassword := username != "" && resolvedType.RequiredFields.Password
			if needsPassword {
				if term.IsTerminal(int(os.Stdin.Fd())) {
					fmt.Fprint(os.Stderr, "Password: ")
					b, err := term.ReadPassword(int(os.Stdin.Fd()))
					fmt.Fprintln(os.Stderr)
					if err == nil {
						password = string(b)
					}
				} else {
					// Non-TTY: only read from stdin when --password is provided
					if passwordFromStdin {
						fi, _ := os.Stdin.Stat()
						if (fi.Mode() & os.ModeCharDevice) == 0 {
							r := bufio.NewReader(os.Stdin)
							line, _ := r.ReadString('\n')
							password = strings.Trim(line, "\r\n")
						}
					} else {
						return fmt.Errorf("stdin is not a TTY. Use --password and pipe the password on stdin, or run interactively without piping")
					}
				}
			}

			advanced, err := connectionopts.ApplySSLSettings(string(resolvedType.ID), nil, connectionopts.SSLSettings{
				Mode:           connectSSLMode,
				CAFile:         connectSSLCA,
				ClientCertFile: connectSSLCert,
				ClientKeyFile:  connectSSLKey,
				ServerName:     connectSSLServerName,
			})
			if err != nil {
				return err
			}

			conn := config.Connection{
				Name:     name,
				Type:     string(resolvedType.ID),
				Host:     host,
				Port:     port,
				Username: username,
				Password: password,
				Database: database,
				Schema:   schema,
				Advanced: advanced,
			}

			if name != "" {
				cfg, err := config.LoadConfig()
				if err != nil {
					return fmt.Errorf("error loading config: %w", err)
				}

				cfg.AddConnection(conn)
				if err := cfg.Save(); err != nil {
					return fmt.Errorf("error saving connection: %w", err)
				}
				fmt.Printf("Connection '%s' saved successfully\n", name)
			}

			m := tui.NewMainModelWithConnection(&conn)
			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("error running interactive mode: %w", err)
			}

			return nil
		}

		// Otherwise, launch TUI with connection form
		m := tui.NewMainModel()
		p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running interactive mode: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)

	connectCmd.Flags().StringVar(&dbType, "type", "", "database type (postgres, mysql, sqlite, duckdb, mongodb, redis, etc.)")
	connectCmd.Flags().StringVar(&host, "host", "", "database host")
	connectCmd.Flags().IntVar(&port, "port", 0, "database port (default depends on database type)")
	connectCmd.Flags().StringVar(&username, "user", "", "database username")
	connectCmd.Flags().StringVar(&database, "database", "", "database name")
	connectCmd.Flags().StringVar(&schema, "schema", "", "preferred schema (PostgreSQL: schema name; MySQL: not needed; MongoDB: not applicable)")
	connectCmd.Flags().StringVar(&name, "name", "", "connection name (save for later use)")
	connectCmd.Flags().BoolVar(&passwordFromStdin, "password", false, "read password from stdin when not using a TTY")
	connectCmd.Flags().BoolVar(&useDocker, "docker", false, "auto-detect running Docker database containers and connect to the first match")
	connectCmd.Flags().StringVar(&connectSSLMode, "ssl-mode", "", "SSL mode from the selected database type's supported modes")
	connectCmd.Flags().StringVar(&connectSSLCA, "ssl-ca", "", "path to a CA certificate PEM file")
	connectCmd.Flags().StringVar(&connectSSLCert, "ssl-cert", "", "path to a client certificate PEM file")
	connectCmd.Flags().StringVar(&connectSSLKey, "ssl-key", "", "path to a client private key PEM file")
	connectCmd.Flags().StringVar(&connectSSLServerName, "ssl-server-name", "", "override server name used for SSL hostname verification")

	connectCmd.RegisterFlagCompletionFunc("type", completeDatabaseTypes)
	connectCmd.RegisterFlagCompletionFunc("ssl-mode", completeSSLModes)
}
