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
	"encoding/json"

	"github.com/clidey/whodb/cli/pkg/output"
	"github.com/spf13/cobra"
)

type automationEnvelope struct {
	Command string `json:"command"`
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
}

func newCommandOutput(cmd *cobra.Command, format output.Format, quiet bool) *output.Writer {
	return output.New(
		output.WithFormat(format),
		output.WithQuiet(quiet),
		output.WithOutput(cmd.OutOrStdout()),
		output.WithErrorOutput(cmd.ErrOrStderr()),
	)
}

func writeCommandJSON(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeEmptyJSONArray(cmd *cobra.Command) error {
	return writeCommandJSON(cmd, []any{})
}

func writeAutomationEnvelope(cmd *cobra.Command, command string, data any) error {
	return writeCommandJSON(cmd, automationEnvelope{
		Command: command,
		Success: true,
		Data:    data,
	})
}
