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

package connectionopts

import (
	"os"
	"path/filepath"
	"testing"

	coressl "github.com/clidey/whodb/core/src/common/ssl"
)

func TestApplySSLSettingsReadsFiles(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	certPath := filepath.Join(dir, "client-cert.pem")
	keyPath := filepath.Join(dir, "client-key.pem")

	if err := os.WriteFile(caPath, []byte("ca-content"), 0o600); err != nil {
		t.Fatalf("write CA file: %v", err)
	}
	if err := os.WriteFile(certPath, []byte("cert-content"), 0o600); err != nil {
		t.Fatalf("write cert file: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("key-content"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	advanced, err := ApplySSLSettings("Postgres", nil, SSLSettings{
		Mode:           "verify-identity",
		CAFile:         caPath,
		ClientCertFile: certPath,
		ClientKeyFile:  keyPath,
		ServerName:     "db.internal",
	})
	if err != nil {
		t.Fatalf("ApplySSLSettings failed: %v", err)
	}

	if advanced[coressl.KeySSLMode] != "verify-identity" {
		t.Fatalf("expected verify-identity mode, got %#v", advanced)
	}
	if advanced[coressl.KeySSLCACertContent] != "ca-content" {
		t.Fatalf("expected CA content, got %#v", advanced)
	}
	if advanced[coressl.KeySSLClientCertContent] != "cert-content" {
		t.Fatalf("expected client cert content, got %#v", advanced)
	}
	if advanced[coressl.KeySSLClientKeyContent] != "key-content" {
		t.Fatalf("expected client key content, got %#v", advanced)
	}
	if advanced[coressl.KeySSLServerName] != "db.internal" {
		t.Fatalf("expected server name, got %#v", advanced)
	}
}

func TestApplySSLSettingsRejectsMissingMode(t *testing.T) {
	_, err := ApplySSLSettings("Postgres", nil, SSLSettings{ServerName: "db.internal"})
	if err == nil {
		t.Fatal("expected error when SSL options are set without a mode")
	}
}

func TestApplySSLSettingsRejectsUnsupportedDatabase(t *testing.T) {
	_, err := ApplySSLSettings("Sqlite3", nil, SSLSettings{Mode: "required"})
	if err == nil {
		t.Fatal("expected error for unsupported SSL database type")
	}
}
