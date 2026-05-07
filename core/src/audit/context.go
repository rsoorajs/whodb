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

package audit

import (
	"context"
	"strings"
)

type requestContextKey struct{}

// WithRequest stores request metadata inside the supplied context.
func WithRequest(ctx context.Context, request Request) context.Context {
	return context.WithValue(ctx, requestContextKey{}, mergeRequest(RequestFromContext(ctx), request))
}

// RequestFromContext extracts request metadata from the supplied context.
func RequestFromContext(ctx context.Context) Request {
	value := ctx.Value(requestContextKey{})
	request, ok := value.(Request)
	if !ok {
		return Request{}
	}
	return request
}

func mergeRequest(base Request, override Request) Request {
	request := base

	if strings.TrimSpace(override.ID) != "" {
		request.ID = strings.TrimSpace(override.ID)
	}
	if strings.TrimSpace(override.Host) != "" {
		request.Host = strings.TrimSpace(override.Host)
	}
	if strings.TrimSpace(override.Method) != "" {
		request.Method = strings.TrimSpace(override.Method)
	}
	if strings.TrimSpace(override.Path) != "" {
		request.Path = strings.TrimSpace(override.Path)
	}
	if strings.TrimSpace(override.Route) != "" {
		request.Route = strings.TrimSpace(override.Route)
	}
	if strings.TrimSpace(override.RemoteIP) != "" {
		request.RemoteIP = strings.TrimSpace(override.RemoteIP)
	}
	if strings.TrimSpace(override.UserAgent) != "" {
		request.UserAgent = strings.TrimSpace(override.UserAgent)
	}
	if strings.TrimSpace(override.Protocol) != "" {
		request.Protocol = strings.TrimSpace(override.Protocol)
	}
	if strings.TrimSpace(override.OperationName) != "" {
		request.OperationName = strings.TrimSpace(override.OperationName)
	}
	if strings.TrimSpace(override.OperationType) != "" {
		request.OperationType = strings.TrimSpace(override.OperationType)
	}
	if strings.TrimSpace(override.TraceID) != "" {
		request.TraceID = strings.TrimSpace(override.TraceID)
	}
	if strings.TrimSpace(override.SpanID) != "" {
		request.SpanID = strings.TrimSpace(override.SpanID)
	}

	return request
}
