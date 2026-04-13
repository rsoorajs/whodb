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

package engine

// globalPlugins holds plugins registered via init() in each plugin package.
// The entry point's blank imports control which plugins are registered.
var globalPlugins []*Plugin

// pluginTypeAliases maps alias database types to their underlying plugin types.
// Populated via RegisterPluginTypeAlias (e.g., by EE init code).
var pluginTypeAliases = map[DatabaseType]DatabaseType{}

// RegisterPlugin adds a plugin to the global registry.
// Called from init() in each plugin package.
func RegisterPlugin(p *Plugin) {
	globalPlugins = append(globalPlugins, p)
}

// RegisteredPlugins returns all plugins registered via RegisterPlugin.
func RegisteredPlugins() []*Plugin {
	return globalPlugins
}

// RegisterPluginTypeAlias registers a mapping from an alias database type to its
// underlying plugin type. This allows extensions (e.g., EE) to add new database
// types that resolve to existing plugins without modifying CE code.
func RegisterPluginTypeAlias(alias DatabaseType, target DatabaseType) {
	pluginTypeAliases[alias] = target
}
