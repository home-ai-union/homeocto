package internal

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestGetConfigPath(t *testing.T) {
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := filepath.Join("/tmp/home", ".homeocto", "config.json")

	assert.Equal(t, want, got)
}

func TestGetConfigPath_WithHOMEOCTO_HOME(t *testing.T) {
	t.Setenv(config.EnvHome, "/custom/homeocto")
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := filepath.Join("/custom/homeocto", "config.json")

	assert.Equal(t, want, got)
}

func TestGetConfigPath_WithHOMEOCTO_CONFIG(t *testing.T) {
	t.Setenv("HOMEOCTO_CONFIG", "/custom/config.json")
	t.Setenv(config.EnvHome, "/custom/homeocto")
	t.Setenv("HOME", "/tmp/home")

	got := GetConfigPath()
	want := "/custom/config.json"

	assert.Equal(t, want, got)
}

func TestGetConfigPath_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific HOME behavior varies; run on windows")
	}

	testUserProfilePath := `C:\Users\Test`
	t.Setenv("USERPROFILE", testUserProfilePath)

	got := GetConfigPath()
	want := filepath.Join(testUserProfilePath, ".homeocto", "config.json")

	require.True(t, strings.EqualFold(got, want), "GetConfigPath() = %q, want %q", got, want)
}
