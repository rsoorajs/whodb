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

// Package whodbplugin embeds the WhoDB assistant plugin assets in the CLI.
package whodbplugin

import "embed"

// FS contains the bundled WhoDB skills, agents, and plugin metadata.
//
//go:embed README.md .mcp.json .claude-plugin/plugin.json skills/*/SKILL.md agents/*.md
var FS embed.FS
