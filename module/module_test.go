package module

import (
	"os"
	"strings"
	"testing"
)

func TestRuntimeSetupHasNoMigrationExecutionPath(t *testing.T) {
	source, err := os.ReadFile("module.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "func setupRuntimeModules(")
	end := strings.Index(text[start:], "// RegisteredMigrationSource")
	if start < 0 || end < 0 {
		t.Fatal("SetupRuntime section missing")
	}
	section := text[start : start+end]
	if strings.Contains(section, "executeSQL(") || strings.Contains(section, "RegisteredMigrationSource") {
		t.Fatal("SetupRuntime must not execute or enumerate schema migrations")
	}
}

func TestRegisteredMigrationSourceDoesNotInstantiateModules(t *testing.T) {
	source, err := os.ReadFile("module.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "func RegisteredMigrationSource(")
	end := strings.Index(text[start:], "func Start(")
	if start < 0 || end < 0 {
		t.Fatal("RegisteredMigrationSource section missing")
	}
	section := text[start : start+end]
	if strings.Contains(section, "GetModules(") || strings.Contains(section, "config.Context") {
		t.Fatal("migration source enumeration must not instantiate runtime modules or contexts")
	}
}
