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

// Package connections provides lightweight connection discovery and resolution
// using saved CLI config and environment profiles without initializing the
// database engine.
package connections

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/sourcetypes"
	"github.com/clidey/whodb/core/src/envconfig"
	"github.com/clidey/whodb/core/src/types"
)

// ConnectionSourceSaved identifies a saved CLI connection.
const ConnectionSourceSaved = "saved"

// ConnectionSourceEnv identifies a connection discovered from environment
// profiles.
const ConnectionSourceEnv = "env"

// ConnectionSourceInfo describes one resolved connection and its origin.
type ConnectionSourceInfo struct {
	Connection config.Connection
	Source     string
}

// Resolver provides lightweight connection lookup against saved config and
// environment profiles.
type Resolver struct {
	infos []ConnectionSourceInfo
}

// NewResolver loads a resolver from disk and environment.
func NewResolver(includeSecrets bool) (*Resolver, error) {
	var (
		cfg *config.Config
		err error
	)

	if includeSecrets {
		cfg, err = config.LoadConfig()
	} else {
		cfg, err = config.LoadConfigWithoutSecrets()
	}
	if err != nil {
		return nil, err
	}

	return NewResolverWithConfig(cfg), nil
}

// NewResolverWithConfig builds a resolver from an already-loaded CLI config.
func NewResolverWithConfig(cfg *config.Config) *Resolver {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &Resolver{
		infos: listConnectionsWithSource(cfg),
	}
}

// Count returns the total number of available connections.
func (r *Resolver) Count() int {
	if r == nil {
		return 0
	}
	return len(r.infos)
}

// ListWithSource returns saved and environment connections, with saved
// connections taking precedence when names collide.
func (r *Resolver) ListWithSource() []ConnectionSourceInfo {
	if r == nil {
		return nil
	}
	infos := make([]ConnectionSourceInfo, len(r.infos))
	copy(infos, r.infos)
	return infos
}

// List returns all available connections without their source metadata.
func (r *Resolver) List() []config.Connection {
	if r == nil {
		return nil
	}
	conns := make([]config.Connection, len(r.infos))
	for i, info := range r.infos {
		conns[i] = info.Connection
	}
	return conns
}

// Resolve finds a connection by name from saved config or environment
// profiles.
func (r *Resolver) Resolve(name string) (*config.Connection, string, error) {
	if strings.TrimSpace(name) == "" {
		return nil, "", fmt.Errorf("connection name is required")
	}
	if r == nil {
		return nil, "", fmt.Errorf("connection %q not found", name)
	}

	for _, info := range r.infos {
		if info.Connection.Name == name {
			conn := info.Connection
			return &conn, info.Source, nil
		}
	}

	return nil, "", fmt.Errorf("connection %q not found", name)
}

// EnvConnections returns all environment-discovered connections.
func EnvConnections() []config.Connection {
	typeCounts := make(map[string]int)
	connections := make([]config.Connection, 0)

	for _, dbType := range sourcetypes.IDs() {
		profiles := envconfig.GetDefaultDatabaseCredentials(dbType)
		for _, profile := range profiles {
			typeCounts[dbType]++
			connections = append(connections, envProfileToConnection(profile, dbType, typeCounts[dbType]))
		}
	}

	return connections
}

func listConnectionsWithSource(cfg *config.Config) []ConnectionSourceInfo {
	saved := cfg.Connections
	envConnections := EnvConnections()

	infos := make([]ConnectionSourceInfo, 0, len(saved)+len(envConnections))
	usedNames := make(map[string]bool, len(saved)+len(envConnections))

	for _, conn := range saved {
		infos = append(infos, ConnectionSourceInfo{
			Connection: conn,
			Source:     ConnectionSourceSaved,
		})
		usedNames[conn.Name] = true
	}

	for _, conn := range envConnections {
		if conn.Name == "" || usedNames[conn.Name] {
			continue
		}
		infos = append(infos, ConnectionSourceInfo{
			Connection: conn,
			Source:     ConnectionSourceEnv,
		})
		usedNames[conn.Name] = true
	}

	return infos
}

func envProfileToConnection(profile types.DatabaseCredentials, dbType string, index int) config.Connection {
	name := envProfileName(profile, dbType, index)

	var advanced map[string]string
	if profile.Port != "" || len(profile.Advanced) > 0 {
		advanced = make(map[string]string, len(profile.Advanced)+1)
		for key, value := range profile.Advanced {
			advanced[key] = value
		}
		if profile.Port != "" {
			advanced["Port"] = profile.Port
		}
	}

	port := 0
	if profile.Port != "" {
		if parsedPort, err := strconv.Atoi(profile.Port); err == nil {
			port = parsedPort
		}
	}

	return config.Connection{
		Name:      name,
		Type:      dbType,
		Host:      profile.Hostname,
		Port:      port,
		Username:  profile.Username,
		Password:  profile.Password,
		Database:  profile.Database,
		Advanced:  advanced,
		IsProfile: true,
	}
}

func envProfileName(profile types.DatabaseCredentials, dbType string, index int) string {
	if profile.CustomId != "" {
		return profile.CustomId
	}
	if profile.Alias != "" {
		return profile.Alias
	}
	return fmt.Sprintf("%s-%d", strings.ToLower(dbType), index)
}
