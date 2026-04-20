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

package clickhouse

import (
	"github.com/clidey/whodb/core/src/common"
	sourcecatalogspecs "github.com/clidey/whodb/core/src/sourcecatalog/specs"
)

// AliasMap maps ClickHouse type aliases to their canonical names.
// Note: ClickHouse uses mixed-case type names (Int8, String, etc.)
var AliasMap = sourcecatalogspecs.ClickHouseAliasMap

// TypeDefinitions contains the canonical ClickHouse types with metadata for UI.
var TypeDefinitions = sourcecatalogspecs.ClickHouseTypeDefinitions

// NormalizeType converts a ClickHouse type alias to its canonical form.
func NormalizeType(typeName string) string {
	return common.NormalizeTypeWithMap(typeName, AliasMap)
}
