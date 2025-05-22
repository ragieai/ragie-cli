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
		Delay:  0,        // No delay for tests
		Mode:   "hi_res", // Test with hi_res mode
	}
	err := cmd.ImportFiles(c, testDir, config)
	if err != nil {
		t.Fatalf("Failed to import files: %v", err)
	}

	// Verify the imports
	t.Log("Verifying imported documents...")
	time.Sleep(1 * time.Second) // Give API some time to process

	// Check first file
	resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": "file1.txt"},
		PageSize: 1,
	})
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
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": "subdir/file3.json"},
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Error("Expected to find subdir/file3.json document")
	}

	// Verify that empty file was not imported
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": "empty.txt"},
		PageSize: 1,
	})
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

func TestFilesImportForce(t *testing.T) {
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

	testFile := "force_test.txt"
	testContent := "This is a test file for force flag"

	// Clean up any existing test documents with this external ID
	if resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testFile},
		PageSize: 100,
	}); err == nil {
		for _, doc := range resp.Documents {
			c.DeleteDocument(doc.ID)
		}
	}

	// Create temporary test directory and file
	tempDir := t.TempDir()
	tempFilePath := filepath.Join(tempDir, testFile)
	if err := os.WriteFile(tempFilePath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// First import without force (import the directory, not the single file)
	t.Log("Running first files import...")
	config := cmd.ImportConfig{
		DryRun: false,
		Delay:  0,
		Force:  false,
		Mode:   "fast",
	}

	err := cmd.ImportFiles(c, tempDir, config)
	if err != nil {
		t.Fatalf("Failed to import files: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify document was created
	resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testFile},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(resp.Documents))
	}

	// Second import without force - should skip
	t.Log("Running second files import without force...")
	err = cmd.ImportFiles(c, tempDir, config)
	if err != nil {
		t.Fatalf("Failed to import files: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify still only one document
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testFile},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Errorf("Expected 1 document after second import without force, got %d", len(resp.Documents))
	}

	// Third import with force - should create duplicate
	t.Log("Running third files import with force...")
	config.Force = true
	err = cmd.ImportFiles(c, tempDir, config)
	if err != nil {
		t.Fatalf("Failed to import files with force: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify now two documents exist
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testFile},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 2 {
		t.Errorf("Expected 2 documents after force import, got %d", len(resp.Documents))
	}

	// Clean up test documents
	if resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testFile},
		PageSize: 100,
	}); err == nil {
		for _, doc := range resp.Documents {
			if err := c.DeleteDocument(doc.ID); err != nil {
				t.Logf("Error deleting document %s: %v", doc.ID, err)
			}
		}
	}
}

func TestFilesImportReplace(t *testing.T) {
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

	testFile := "replace_test.txt"
	testContent1 := "This is the first version of the test file"
	testContent2 := "This is the second version of the test file"

	// Clean up any existing test documents with this external ID
	if resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testFile},
		PageSize: 100,
	}); err == nil {
		for _, doc := range resp.Documents {
			c.DeleteDocument(doc.ID)
		}
	}

	// Create temporary test directory and file with first content
	tempDir := t.TempDir()
	tempFilePath := filepath.Join(tempDir, testFile)
	if err := os.WriteFile(tempFilePath, []byte(testContent1), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// First import - create initial document
	t.Log("Running first files import...")
	config := cmd.ImportConfig{
		DryRun:  false,
		Delay:   0,
		Force:   false,
		Replace: false,
		Mode:    "fast",
	}

	err := cmd.ImportFiles(c, tempDir, config)
	if err != nil {
		t.Fatalf("Failed to import files: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify document was created
	resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testFile},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(resp.Documents))
	}
	firstDocID := resp.Documents[0].ID

	// Update file content for replacement
	if err := os.WriteFile(tempFilePath, []byte(testContent2), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// Second import with replace - should replace the existing document
	t.Log("Running second files import with replace...")
	config.Replace = true
	err = cmd.ImportFiles(c, tempDir, config)
	if err != nil {
		t.Fatalf("Failed to import files with replace: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify still only one document but with different ID (replaced)
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testFile},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Errorf("Expected 1 document after replace, got %d", len(resp.Documents))
	}

	// Verify that the document ID changed (old one was deleted, new one created)
	if len(resp.Documents) > 0 && resp.Documents[0].ID == firstDocID {
		t.Errorf("Document ID should have changed after replace, but got same ID: %s", firstDocID)
	}

	// Clean up test documents
	if resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testFile},
		PageSize: 100,
	}); err == nil {
		for _, doc := range resp.Documents {
			if err := c.DeleteDocument(doc.ID); err != nil {
				t.Logf("Error deleting document %s: %v", doc.ID, err)
			}
		}
	}
}

func cleanupFilesTestDocuments(t *testing.T, c *client.Client) {
	testFiles := []string{
		"file1.txt",
		"file2.md",
		"subdir/file3.json",
		"empty.txt",
	}

	for _, path := range testFiles {
		resp, err := c.ListDocuments(client.ListOptions{
			Filter:   map[string]interface{}{"external_id": path},
			PageSize: 1,
		})
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
