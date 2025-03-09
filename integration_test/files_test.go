package integration_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ragie/cmd"
	"ragie/pkg/client"

	"github.com/spf13/viper"
)

func TestFilesImport(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	// Check for API key
	apiKey := os.Getenv("RAGIE_API_KEY")
	if apiKey == "" {
		t.Fatal("RAGIE_API_KEY environment variable must be set")
	}

	// Initialize the client
	c := client.NewClient(apiKey)
	viper.Set("api_key", apiKey)

	// Create a temporary test directory
	testDir := filepath.Join(t.TempDir(), "test_files")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test files
	testFiles := map[string]string{
		"file1.txt":         "This is test file 1",
		"file2.md":          "# Test File 2\nThis is a markdown file",
		"subdir/file3.json": `{"key": "value"}`,
		"empty.txt":         "",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(testDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	// Clean up any existing test documents
	t.Log("Cleaning up existing test documents...")
	cleanupFilesTestDocuments(t, c)

	// Run the import
	t.Log("Running files import...")
	config := cmd.ImportConfig{
		DryRun: false,
		Delay:  0, // No delay for tests
	}
	err := cmd.ImportFiles(c, testDir, config)
	if err != nil {
		t.Fatalf("Failed to import files: %v", err)
	}

	// Verify the imports
	t.Log("Verifying imported documents...")
	time.Sleep(1 * time.Second) // Give API some time to process

	// Check first file
	resp, err := c.ListDocuments("", map[string]interface{}{"external_id": "file1.txt"}, 1)
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Error("Expected to find file1.txt document")
	} else {
		doc := resp.Documents[0]
		if doc.Name != "file1.txt" {
			t.Errorf("Expected name 'file1.txt', got '%s'", doc.Name)
		}
		if doc.Metadata["source_type"] != "files" {
			t.Errorf("Expected source_type 'files', got '%v'", doc.Metadata["source_type"])
		}
		if doc.Metadata["extension"] != ".txt" {
			t.Errorf("Expected extension '.txt', got '%v'", doc.Metadata["extension"])
		}
	}

	// Check nested file
	resp, err = c.ListDocuments("", map[string]interface{}{"external_id": "subdir/file3.json"}, 1)
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Error("Expected to find subdir/file3.json document")
	}

	// Verify that empty file was not imported
	resp, err = c.ListDocuments("", map[string]interface{}{"external_id": "empty.txt"}, 1)
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 0 {
		t.Error("Empty file should not have been imported")
	}

	// Clean up test documents
	t.Log("Cleaning up test documents...")
	cleanupFilesTestDocuments(t, c)
}

func cleanupFilesTestDocuments(t *testing.T, c *client.Client) {
	testFiles := []string{
		"file1.txt",
		"file2.md",
		"subdir/file3.json",
		"empty.txt",
	}

	for _, path := range testFiles {
		resp, err := c.ListDocuments("", map[string]interface{}{"external_id": path}, 1)
		if err != nil {
			t.Logf("Error listing documents for cleanup: %v", err)
			continue
		}
		for _, doc := range resp.Documents {
			if err := c.DeleteDocument(doc.ID); err != nil {
				t.Logf("Error deleting document %s: %v", doc.ID, err)
			}
		}
	}
}
