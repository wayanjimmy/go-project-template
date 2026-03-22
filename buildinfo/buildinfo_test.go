package buildinfo

import (
	"runtime"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()

	if info.Version == "" {
		t.Error("Version should not be empty")
	}
	if info.Service == "" {
		t.Error("Service should not be empty")
	}
	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
	if info.Platform == "" {
		t.Error("Platform should not be empty")
	}

	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if info.Platform != expectedPlatform {
		t.Errorf("Platform = %q, want %q", info.Platform, expectedPlatform)
	}
}

func TestGetDefaultValues(t *testing.T) {
	info := Get()

	if info.Version != "dev" {
		t.Errorf("Version = %q, want %q", info.Version, "dev")
	}
	if info.Service != "unknown" {
		t.Errorf("Service = %q, want %q", info.Service, "unknown")
	}
}

func TestString(t *testing.T) {
	s := String()

	if s == "" {
		t.Error("String() should not return empty string")
	}

	expectedFields := []string{"service=", "version=", "go="}
	for _, field := range expectedFields {
		if !strings.Contains(s, field) {
			t.Errorf("String() should contain %q", field)
		}
	}
}

func TestStringContainsRuntimeVersion(t *testing.T) {
	s := String()
	if !strings.Contains(s, runtime.Version()) {
		t.Errorf("String() should contain runtime version %q", runtime.Version())
	}
}
