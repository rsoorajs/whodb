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

func TestApplicationVersionDefaultsToDev(t *testing.T) {
	originalVersion := env.ApplicationVersion
	t.Cleanup(func() {
		env.ApplicationVersion = originalVersion
	})

	env.ApplicationVersion = ""
	if got := applicationVersion(); got != "dev" {
		t.Fatalf("expected default application version, got %q", got)
	}

	env.ApplicationVersion = "2.0.0"
	if got := applicationVersion(); got != "2.0.0" {
		t.Fatalf("expected configured application version, got %q", got)
	}
}

func TestRuntimeModePrefersDesktopThenCLI(t *testing.T) {
	t.Setenv("WHODB_DESKTOP", "")
	t.Setenv("WHODB_CLI", "")
	if got := runtimeMode(); got != "server" {
		t.Fatalf("expected server mode by default, got %q", got)
	}

	t.Setenv("WHODB_CLI", "true")
	if got := runtimeMode(); got != "cli" {
		t.Fatalf("expected cli mode, got %q", got)
	}

	t.Setenv("WHODB_DESKTOP", "true")
	if got := runtimeMode(); got != "desktop" {
		t.Fatalf("expected desktop mode to take precedence, got %q", got)
	}
}

func TestRegisteredPluginNamesIncludesRegisteredTypes(t *testing.T) {
	testType := engine.DatabaseType("RegisteredPluginNamesTest")
	engine.RegisterPlugin(&engine.Plugin{Type: testType})

	pluginNames := registeredPluginNames()
	if len(pluginNames) != len(engine.RegisteredPlugins()) {
		t.Fatalf("expected plugin names to match registered plugins, got %d want %d", len(pluginNames), len(engine.RegisteredPlugins()))
	}

	if pluginNames[len(pluginNames)-1] != string(testType) {
		t.Fatalf("expected last plugin name to be %q, got %#v", testType, pluginNames)
	}
}

func TestResolvePortUsesEnvironmentOverride(t *testing.T) {
	t.Setenv("PORT", "")
	if got := resolvePort(); got != defaultPort {
		t.Fatalf("expected default port, got %q", got)
	}

	t.Setenv("PORT", "9090")
	if got := resolvePort(); got != "9090" {
		t.Fatalf("expected overridden port, got %q", got)
	}
}

func TestWelcomeBannerLinesReflectEditionAndPort(t *testing.T) {
	originalEE := env.IsEnterpriseEdition
	t.Cleanup(func() {
		env.IsEnterpriseEdition = originalEE
	})

	env.IsEnterpriseEdition = false
	if got := welcomeBannerLines("8080"); strings.Join(got, "\n") != strings.Join([]string{
		"🎉 Welcome to WhoDB! 🎉",
		"Get started by visiting:",
		"http://0.0.0.0:8080",
		"Explore and enjoy working with your databases!",
	}, "\n") {
		t.Fatalf("unexpected CE welcome banner: %#v", got)
	}

	env.IsEnterpriseEdition = true
	got := welcomeBannerLines("9090")
	if got[0] != "🎉 Welcome to WhoDB Enterprise! 🎉" {
		t.Fatalf("expected EE banner title, got %q", got[0])
	}
	if got[2] != "http://0.0.0.0:9090" {
		t.Fatalf("expected banner URL to include configured port, got %q", got[2])
	}
}
