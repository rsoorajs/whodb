//go:build !arm && !riscv64

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

// AgentHandlerFunc is the type for the agent stream handler.
type AgentHandlerFunc func(w http.ResponseWriter, r *http.Request)

var registeredAgentHandler AgentHandlerFunc
var registeredAgentPermitHandler AgentHandlerFunc

// RegisterAgentHandler registers the agent stream handler.
func RegisterAgentHandler(handler AgentHandlerFunc) {
	registeredAgentHandler = handler
}

// RegisterAgentPermitHandler registers the agent permission handler.
func RegisterAgentPermitHandler(handler AgentHandlerFunc) {
	registeredAgentPermitHandler = handler
}

// agentStreamHandler delegates to the registered implementation.
func agentStreamHandler(w http.ResponseWriter, r *http.Request) {
	if registeredAgentHandler != nil {
		registeredAgentHandler(w, r)
		return
	}
	http.Error(w, "Agent not available in this edition", http.StatusNotImplemented)
}

// agentPermitHandler delegates to the registered implementation.
func agentPermitHandler(w http.ResponseWriter, r *http.Request) {
	if registeredAgentPermitHandler != nil {
		registeredAgentPermitHandler(w, r)
		return
	}
	http.Error(w, "Agent not available in this edition", http.StatusNotImplemented)
}
