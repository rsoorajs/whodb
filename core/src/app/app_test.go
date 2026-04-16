package app

import (
	"embed"
	"flag"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
)

func TestPopulateActiveDatabasesAppendsRegisteredPlugins(t *testing.T) {
	originalActive := append([]string(nil), env.ActiveDatabases...)
	t.Cleanup(func() {
		env.ActiveDatabases = originalActive
	})

	testType := engine.DatabaseType("PopulateActiveDatabasesTest")
	engine.RegisterPlugin(&engine.Plugin{Type: testType})
	env.ActiveDatabases = nil

	PopulateActiveDatabases()

	if len(env.ActiveDatabases) != len(engine.RegisteredPlugins()) {
		t.Fatalf("expected active database count to match registered plugins, got %d want %d", len(env.ActiveDatabases), len(engine.RegisteredPlugins()))
	}
	if env.ActiveDatabases[len(env.ActiveDatabases)-1] != string(testType) {
		t.Fatalf("expected test plugin type to be appended, got %#v", env.ActiveDatabases)
	}
}

func TestRunPrintsVersionAndReturns(t *testing.T) {
	originalFlagSet := flag.CommandLine
	originalArgs := os.Args
	originalStdout := os.Stdout
	originalVersion := env.ApplicationVersion
	t.Cleanup(func() {
		flag.CommandLine = originalFlagSet
		os.Args = originalArgs
		os.Stdout = originalStdout
		env.ApplicationVersion = originalVersion
	})

	flag.CommandLine = flag.NewFlagSet("whodb-test", flag.ContinueOnError)
	os.Args = []string{"whodb-test", "-version"}
	env.ApplicationVersion = "1.2.3"

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = writer

	Run(AppConfig{}, embed.FS{})

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close stdout writer: %v", err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}

	if got := strings.TrimSpace(string(output)); got != "1.2.3" {
		t.Fatalf("expected version output, got %q", got)
	}
}
