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

package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/clidey/whodb/core/src/engine"
)

// AddRowFromJSON inserts a new row/document into a writable storage unit using
// a JSON object payload.
func (m *Manager) AddRowFromJSON(schema, storageUnit string, payload string) error {
	if m.currentConnection == nil {
		return fmt.Errorf("not connected to any database")
	}
	if m.config != nil && m.config.GetReadOnly() {
		return ErrReadOnly
	}
	if err := m.ensureWritableStorageUnit(schema, storageUnit); err != nil {
		return err
	}

	values, err := parseRowPayload(payload)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return fmt.Errorf("row payload must contain at least one field")
	}

	plugin, config, err := m.currentPlugin()
	if err != nil {
		return err
	}

	columns, err := m.GetColumns(schema, storageUnit)
	if err != nil {
		return err
	}

	records, err := buildRowRecords(values, columns)
	if err != nil {
		return err
	}

	_, err = plugin.AddRow(config, schema, storageUnit, records)
	if err != nil {
		return err
	}

	if m.cache != nil {
		m.cache.Clear()
	}
	return nil
}

// DeleteRow deletes a row/document from a writable storage unit.
func (m *Manager) DeleteRow(schema, storageUnit string, values map[string]string) error {
	if m.currentConnection == nil {
		return fmt.Errorf("not connected to any database")
	}
	if m.config != nil && m.config.GetReadOnly() {
		return ErrReadOnly
	}
	if err := m.ensureWritableStorageUnit(schema, storageUnit); err != nil {
		return err
	}
	if len(values) == 0 {
		return fmt.Errorf("delete requires at least one row value")
	}

	plugin, config, err := m.currentPlugin()
	if err != nil {
		return err
	}

	_, err = plugin.DeleteRow(config, schema, storageUnit, values)
	if err != nil {
		return err
	}

	if m.cache != nil {
		m.cache.Clear()
	}
	return nil
}

func parseRowPayload(payload string) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewBufferString(payload))
	decoder.UseNumber()

	var values map[string]any
	if err := decoder.Decode(&values); err != nil {
		return nil, fmt.Errorf("row payload must be a valid JSON object: %w", err)
	}
	if values == nil {
		return nil, fmt.Errorf("row payload must be a JSON object")
	}
	return values, nil
}

func buildRowRecords(values map[string]any, columns []engine.Column) ([]engine.Record, error) {
	columnMap := make(map[string]engine.Column, len(columns))
	for _, column := range columns {
		columnMap[column.Name] = column
	}

	records := make([]engine.Record, 0, len(values))
	for key, rawValue := range values {
		column, knownColumn := columnMap[key]
		if knownColumn && (column.IsComputed || column.IsAutoIncrement) {
			return nil, fmt.Errorf("column %s is database-managed and cannot be inserted explicitly", key)
		}

		value, isNull, err := stringifyRowValue(rawValue)
		if err != nil {
			return nil, fmt.Errorf("invalid value for column %s: %w", key, err)
		}

		record := engine.Record{
			Key:   key,
			Value: value,
			Extra: map[string]string{},
		}
		if knownColumn {
			record.Extra["Type"] = column.Type
			if column.IsNullable {
				record.Extra["IsNullable"] = "true"
			}
		}
		if isNull {
			record.Extra["IsNull"] = "true"
		}

		records = append(records, record)
	}

	return records, nil
}

func stringifyRowValue(value any) (string, bool, error) {
	switch typed := value.(type) {
	case nil:
		return "", true, nil
	case string:
		return typed, false, nil
	case bool:
		return strconv.FormatBool(typed), false, nil
	case json.Number:
		return typed.String(), false, nil
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), false, nil
	case []any, map[string]any:
		data, err := json.Marshal(typed)
		if err != nil {
			return "", false, err
		}
		return string(data), false, nil
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return "", false, err
		}
		return string(data), false, nil
	}
}
