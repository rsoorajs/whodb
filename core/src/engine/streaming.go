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

// QueryStreamWriter receives streamed raw-query output row by row.
// Implementations are expected to handle WriteColumns once before any rows.
type QueryStreamWriter interface {
	WriteColumns(columns []Column) error
	WriteRow(row []string) error
}

// QueryStreamer is implemented by plugins that can stream raw query results
// without materializing the full result set in memory first.
type QueryStreamer interface {
	StreamRawExecute(config *PluginConfig, query string, writer QueryStreamWriter, params ...any) error
}
