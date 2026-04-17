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
	"strings"

	dbmgr "github.com/clidey/whodb/cli/internal/database"
	"github.com/clidey/whodb/core/src/dbcatalog"
)

func resolveSnapshotSchema(mgr *dbmgr.Manager, conn *dbmgr.Connection, explicitSchema string) (string, error) {
	if strings.TrimSpace(explicitSchema) != "" {
		return explicitSchema, nil
	}
	if entry, ok := dbcatalog.Find(conn.Type); ok && entry.UsesDatabaseInsteadOfSchema && strings.TrimSpace(conn.Database) != "" {
		return conn.Database, nil
	}
	if strings.TrimSpace(conn.Schema) != "" {
		return conn.Schema, nil
	}

	schemas, err := mgr.GetSchemas()
	if err != nil || len(schemas) == 0 {
		return "", nil
	}

	return schemas[0], nil
}
