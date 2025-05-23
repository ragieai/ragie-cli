package integration_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ragie/cmd"
	"ragie/pkg/client"

	"github.com/spf13/viper"
)

func TestZipImport(t *testing.T) {
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

	// Create a temporary test directory and zip file
	testDir := t.TempDir()
	zipPath := filepath.Join(testDir, "test_archive.zip")

	// Create a zip file with test content
	if err := createTestZipFile(zipPath); err != nil {
		t.Fatalf("Failed to create test zip file: %v", err)
	}

	// Clean up any existing test documents
	t.Log("Cleaning up existing test documents...")
	cleanupZipTestDocuments(t, c)

	// Run the import
	t.Log("Running zip import...")
	config := cmd.ImportConfig{
		DryRun: false,
		Delay:  0,      // No delay for tests
		Mode:   "fast", // Test with fast mode
	}
	err := cmd.ImportZip(c, zipPath, config)
	if err != nil {
		t.Fatalf("Failed to import zip: %v", err)
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
		if doc.Metadata["source_type"] != "zip" {
			t.Errorf("Expected source_type 'zip', got '%v'", doc.Metadata["source_type"])
		}
		if doc.Metadata["extension"] != ".txt" {
			t.Errorf("Expected extension '.txt', got '%v'", doc.Metadata["extension"])
		}
		if doc.Metadata["zip_source"] != "test_archive.zip" {
			t.Errorf("Expected zip_source 'test_archive.zip', got '%v'", doc.Metadata["zip_source"])
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
	} else {
		doc := resp.Documents[0]
		if doc.Name != "file3.json" {
			t.Errorf("Expected name 'file3.json', got '%s'", doc.Name)
		}
	}

	// Clean up test documents
	t.Log("Cleaning up test documents...")
	cleanupZipTestDocuments(t, c)
}

func TestZipImportForce(t *testing.T) {
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

	// Clean up any existing test documents with this external ID
	if resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testFile},
		PageSize: 100,
	}); err == nil {
		for _, doc := range resp.Documents {
			c.DeleteDocument(doc.ID)
		}
	}

	// Create temporary zip file
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "force_test_archive.zip")

	// Create ZIP file with test content
	err := createZipForceTestFile(zipPath, testFile, "This is a test file for force flag")
	if err != nil {
		t.Fatalf("Failed to create test zip file: %v", err)
	}

	// First import without force
	t.Log("Running first zip import...")
	config := cmd.ImportConfig{
		DryRun: false,
		Delay:  0,
		Force:  false,
		Mode:   "fast",
	}

	err = cmd.ImportZip(c, zipPath, config)
	if err != nil {
		t.Fatalf("Failed to import zip: %v", err)
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
	t.Log("Running second zip import without force...")
	err = cmd.ImportZip(c, zipPath, config)
	if err != nil {
		t.Fatalf("Failed to import zip: %v", err)
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
	t.Log("Running third zip import with force...")
	config.Force = true
	err = cmd.ImportZip(c, zipPath, config)
	if err != nil {
		t.Fatalf("Failed to import zip with force: %v", err)
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

// createZipForceTestFile creates a simple zip file with one test file for force testing
func createZipForceTestFile(zipPath, fileName, content string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	f, err := zipWriter.Create(fileName)
	if err != nil {
		return err
	}

	_, err = f.Write([]byte(content))
	return err
}

// createTestZipFile creates a zip file with test content
func createTestZipFile(zipPath string) error {
	// Create a new zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// Create a zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Test files to add to the zip
	testFiles := map[string]string{
		"file1.txt":         "This is test file 1",
		"file2.md":          "# Test File 2\nThis is a markdown file",
		"subdir/file3.json": `{"key": "value"}`,
		"empty.txt":         "",
	}

	// Add files to the zip
	for path, content := range testFiles {
		// Create a new file in the zip
		f, err := zipWriter.Create(path)
		if err != nil {
			return err
		}

		// Write content to the file
		_, err = f.Write([]byte(content))
		if err != nil {
			return err
		}
	}

	return nil
}

// cleanupZipTestDocuments removes test documents created by the zip importer
func cleanupZipTestDocuments(t *testing.T, c *client.Client) {
	// List of test document IDs to clean up
	testIDs := []string{
		"file1.txt",
		"file2.md",
		"subdir/file3.json",
		"empty.txt",
	}

	for _, id := range testIDs {
		resp, err := c.ListDocuments(client.ListOptions{
			Filter:   map[string]interface{}{"external_id": id},
			PageSize: 1,
		})
		if err != nil {
			t.Logf("Error listing document %s: %v", id, err)
			continue
		}

		if len(resp.Documents) > 0 {
			doc := resp.Documents[0]
			err = c.DeleteDocument(doc.ID)
			if err != nil {
				t.Logf("Error deleting document %s: %v", id, err)
			} else {
				t.Logf("Deleted test document: %s", id)
			}
		}
	}
}
