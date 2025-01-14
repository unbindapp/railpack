package mise

import (
	"os"
	"strings"
	"testing"
)

func TestMistGetLatestVersion(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mise-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mise, err := New(tempDir)
	if err != nil {
		t.Fatalf("failed to create mise: %v", err)
	}

	t.Run("node version", func(t *testing.T) {
		nodeVersion, err := mise.GetLatestVersion("node", "22")
		if err != nil {
			t.Fatalf("failed to get latest version: %v", err)
		}

		if !strings.HasPrefix(nodeVersion, "22") {
			t.Errorf("Expected Node.js version to start with 22, got %s", nodeVersion)
		}
	})

	// Test Bun version
	t.Run("bun version", func(t *testing.T) {
		bunVersion, err := mise.GetLatestVersion("bun", "latest")
		if err != nil {
			t.Errorf("Failed to get Bun version: %v", err)
		}
		if bunVersion == "" {
			t.Error("Expected non-empty Bun version")
		}
	})

	// Test non-existent version
	t.Run("non-existent version", func(t *testing.T) {
		_, err := mise.GetLatestVersion("node", "999")
		if err == nil {
			t.Error("Expected error for non-existent version, got nil")
		}
	})
}
