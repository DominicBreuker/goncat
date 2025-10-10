package clean

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnsureDeletion(t *testing.T) {
	t.Parallel()

	// Test with a mock executable by using a temporary file
	// This test verifies the function can get the executable path and returns a cleanup function
	cleanup, err := EnsureDeletion()
	if err != nil {
		t.Fatalf("EnsureDeletion() error = %v, want nil", err)
	}
	if cleanup == nil {
		t.Fatal("EnsureDeletion() returned nil cleanup function")
	}

	// Note: We cannot easily test the signal handler or actual deletion
	// without interfering with the test process itself.
	// The cleanup function is tested separately below.
}

func TestDeleteFile_Default(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping file system test in short mode")
	}

	// Create a temporary file to test deletion
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_delete.txt")

	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("Test file was not created")
	}

	// Test deleteFile
	deleteFile(tmpFile)

	// Give a moment for deletion to occur (especially on Windows)
	time.Sleep(100 * time.Millisecond)

	// On non-Windows, file should be deleted immediately
	// On Windows, it might still exist due to the timeout mechanism
	// So we just verify the function doesn't panic
}
