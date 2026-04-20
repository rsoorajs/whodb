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

// Package cloudprefill defines the shared CLI prefill rules used when a cloud-
// discovered resource is converted into a connectable CLI connection.
package cloudprefill

import (
	"strings"
	"sync"
)

// Rule converts discovered metadata into Advanced connection settings for one
// connectable database type.
type Rule func(meta func(string) string) map[string]string

var (
	rulesMu sync.RWMutex
	rules   = cloneRules(baseRules)
)

var baseRules = map[string]Rule{
	"Postgres": func(meta func(string) string) map[string]string {
		return map[string]string{"SSL Mode": "require"}
	},
	"MySQL": func(meta func(string) string) map[string]string {
		return map[string]string{"SSL Mode": "require"}
	},
	"MariaDB": func(meta func(string) string) map[string]string {
		return map[string]string{"SSL Mode": "require"}
	},
	"ElastiCache": func(meta func(string) string) map[string]string {
		if meta("transitEncryption") == "true" {
			return map[string]string{"TLS": "true"}
		}
		return nil
	},
	"Valkey": func(meta func(string) string) map[string]string {
		if meta("transitEncryption") == "true" {
			return map[string]string{"TLS": "true"}
		}
		return nil
	},
	"DocumentDB": func(meta func(string) string) map[string]string {
		return map[string]string{
			"URL Params": "?tls=true&tlsInsecure=true&replicaSet=rs0&retryWrites=false&readPreference=secondaryPreferred",
		}
	},
	"Redis": func(meta func(string) string) map[string]string {
		return map[string]string{"TLS": "true"}
	},
	"MongoDB": func(meta func(string) string) map[string]string {
		return map[string]string{
			"URL Params": "?tls=true&tlsInsecure=true&retryWrites=false",
		}
	},
}

// RegisterRules merges additional database-specific prefill rules into the
// shared rule set. Later registrations replace earlier rules for the same
// database type.
func RegisterRules(extra map[string]Rule) {
	if len(extra) == 0 {
		return
	}

	rulesMu.Lock()
	defer rulesMu.Unlock()

	merged := cloneRules(rules)
	for key, rule := range extra {
		if strings.TrimSpace(key) == "" || rule == nil {
			continue
		}
		merged[key] = rule
	}
	rules = merged
}

// BuildAdvanced returns the Advanced connection settings implied by the
// discovered metadata for the provided database type.
func BuildAdvanced(databaseType string, metadata map[string]string) map[string]string {
	meta := func(key string) string {
		return strings.TrimSpace(metadata[key])
	}

	advanced := map[string]string{}

	rulesMu.RLock()
	rule := rules[strings.TrimSpace(databaseType)]
	rulesMu.RUnlock()

	if rule != nil {
		for key, value := range rule(meta) {
			if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
				continue
			}
			advanced[key] = value
		}
	}

	if meta("endpointType") == "proxy" && meta("requireTLS") == "true" {
		advanced["SSL Mode"] = "require"
	}

	if len(advanced) == 0 {
		return nil
	}

	return advanced
}

func cloneRules(src map[string]Rule) map[string]Rule {
	if len(src) == 0 {
		return nil
	}

	dst := make(map[string]Rule, len(src))
	for key, rule := range src {
		dst[key] = rule
	}
	return dst
}
