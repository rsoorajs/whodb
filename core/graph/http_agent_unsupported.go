//go:build arm || riscv64

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

package graph

import "net/http"

// agentStreamHandler returns not-implemented on unsupported platforms.
func agentStreamHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Agent not available on this platform", http.StatusNotImplemented)
}

// agentPermitHandler returns not-implemented on unsupported platforms.
func agentPermitHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Agent not available on this platform", http.StatusNotImplemented)
}
