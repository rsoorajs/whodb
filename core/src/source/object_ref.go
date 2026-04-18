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

package source

import (
	"encoding/base64"
	"encoding/json"
	"slices"
)

type objectRefLocatorPayload struct {
	Kind ObjectKind `json:"kind"`
	Path []string   `json:"path"`
}

// NewObjectRef constructs one source object reference with a stable opaque
// locator and a cloned display path.
func NewObjectRef(kind ObjectKind, path []string) ObjectRef {
	ref := ObjectRef{
		Kind: kind,
		Path: slices.Clone(path),
	}
	ref.Locator = EncodeObjectRefLocator(ref)
	return ref
}

// EncodeObjectRefLocator encodes one source object reference into an opaque
// locator string suitable for round-tripping through GraphQL clients.
func EncodeObjectRefLocator(ref ObjectRef) string {
	payload, err := json.Marshal(objectRefLocatorPayload{
		Kind: ref.Kind,
		Path: slices.Clone(ref.Path),
	})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

// DecodeObjectRefLocator decodes one previously encoded source object locator.
func DecodeObjectRefLocator(locator string) (ObjectRef, error) {
	if locator == "" {
		return ObjectRef{}, nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(locator)
	if err != nil {
		return ObjectRef{}, err
	}

	var payload objectRefLocatorPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ObjectRef{}, err
	}

	return ObjectRef{
		Kind:    payload.Kind,
		Path:    slices.Clone(payload.Path),
		Locator: locator,
	}, nil
}

// NormalizeObjectRef ensures that a source object reference always carries both
// an opaque locator and a cloned display path.
func NormalizeObjectRef(ref ObjectRef) ObjectRef {
	normalized := ObjectRef{
		Kind:    ref.Kind,
		Path:    slices.Clone(ref.Path),
		Locator: ref.Locator,
	}

	if len(normalized.Path) == 0 && normalized.Locator != "" {
		if decoded, err := DecodeObjectRefLocator(normalized.Locator); err == nil {
			if normalized.Kind == "" {
				normalized.Kind = decoded.Kind
			}
			normalized.Path = decoded.Path
		}
	}

	if normalized.Locator == "" {
		normalized.Locator = EncodeObjectRefLocator(normalized)
	}

	return normalized
}
