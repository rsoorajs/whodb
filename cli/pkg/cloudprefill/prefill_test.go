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

package cloudprefill

import "testing"

func TestBuildAdvanced_AppliesBaseRules(t *testing.T) {
	advanced := BuildAdvanced("Postgres", map[string]string{})
	if advanced["SSL Mode"] != "require" {
		t.Fatalf("expected Postgres SSL Mode=require, got %#v", advanced)
	}

	advanced = BuildAdvanced("ElastiCache", map[string]string{"transitEncryption": "true"})
	if advanced["TLS"] != "true" {
		t.Fatalf("expected ElastiCache TLS=true, got %#v", advanced)
	}

	advanced = BuildAdvanced("DocumentDB", map[string]string{})
	if advanced["URL Params"] == "" {
		t.Fatalf("expected DocumentDB URL Params to be set, got %#v", advanced)
	}
}

func TestBuildAdvanced_ProxyRequireTLSForcesSSL(t *testing.T) {
	advanced := BuildAdvanced("Postgres", map[string]string{
		"endpointType": "proxy",
		"requireTLS":   "true",
	})

	if advanced["SSL Mode"] != "require" {
		t.Fatalf("expected proxy requireTLS to force SSL Mode=require, got %#v", advanced)
	}
}
