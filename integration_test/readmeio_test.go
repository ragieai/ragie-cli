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

func TestReadmeIOImport(t *testing.T) {
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

	// Clean up any existing test documents
	t.Log("Cleaning up existing test documents...")
	cleanupReadmeIOTestDocuments(t, c)

	// Run the import
	t.Log("Running ReadmeIO import...")
	config := cmd.ImportConfig{
		DryRun: false,
		Delay:  0, // No delay for tests
	}
	err := cmd.ImportReadmeIO(c, "../testdata/readme_sample.zip", config)
	if err != nil {
		t.Fatalf("Failed to import ReadmeIO data: %v", err)
	}

	// Verify the imports
	t.Log("Verifying imported documents...")
	time.Sleep(1 * time.Second) // Give API some time to process

	// Check first document
	resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": "first-doc"},
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Error("Expected to find first document")
	} else {
		doc := resp.Documents[0]
		if doc.Name != "First Test Document" {
			t.Errorf("Expected title 'First Test Document', got '%s'", doc.Name)
		}
		if doc.Metadata["sourceType"] != "readmeio" {
			t.Errorf("Expected sourceType 'readmeio', got '%v'", doc.Metadata["sourceType"])
		}
		if doc.Metadata["category"] != "Getting Started" {
			t.Errorf("Expected category 'Getting Started', got '%v'", doc.Metadata["category"])
		}
	}

	// Check second document
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": "second-doc"},
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Error("Expected to find second document")
	} else {
		doc := resp.Documents[0]
		if doc.Metadata["category"] != "API Reference" {
			t.Errorf("Expected category 'API Reference', got '%v'", doc.Metadata["category"])
		}
	}

	// Verify that invalid document was not imported
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"title": "Invalid Document"},
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 0 {
		t.Error("Invalid document (without frontmatter) should not have been imported")
	}

	// Clean up test documents
	t.Log("Cleaning up test documents...")
	cleanupReadmeIOTestDocuments(t, c)
}

func TestReadmeIOImportForce(t *testing.T) {
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

	testSlug := "force-test-doc"

	// Clean up any existing test documents with this external ID
	if resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testSlug},
		PageSize: 100,
	}); err == nil {
		for _, doc := range resp.Documents {
			c.DeleteDocument(doc.ID)
		}
	}

	// Create temporary README.io ZIP file
	tempDir := t.TempDir()
	tempZip := filepath.Join(tempDir, "readme_force_test.zip")

	// Create markdown file content
	markdownContent := `---
slug: ` + testSlug + `
title: Force Test Document
---

# Force Test Document

This is test content for force flag testing.`

	// Create ZIP file
	err := createReadmeTestZipFile(tempZip, map[string]string{
		"test.md": markdownContent,
	})
	if err != nil {
		t.Fatalf("Failed to create test ZIP: %v", err)
	}

	// First import without force
	t.Log("Running first ReadmeIO import...")
	config := cmd.ImportConfig{
		DryRun: false,
		Delay:  0,
		Force:  false,
	}

	err = cmd.ImportReadmeIO(c, tempZip, config)
	if err != nil {
		t.Fatalf("Failed to import ReadmeIO data: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify document was created
	resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testSlug},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(resp.Documents))
	}

	// Second import without force - should skip
	t.Log("Running second ReadmeIO import without force...")
	err = cmd.ImportReadmeIO(c, tempZip, config)
	if err != nil {
		t.Fatalf("Failed to import ReadmeIO data: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify still only one document
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testSlug},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Errorf("Expected 1 document after second import without force, got %d", len(resp.Documents))
	}

	// Third import with force - should create duplicate
	t.Log("Running third ReadmeIO import with force...")
	config.Force = true
	err = cmd.ImportReadmeIO(c, tempZip, config)
	if err != nil {
		t.Fatalf("Failed to import ReadmeIO data with force: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify now two documents exist
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testSlug},
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
		Filter:   map[string]interface{}{"external_id": testSlug},
		PageSize: 100,
	}); err == nil {
		for _, doc := range resp.Documents {
			if err := c.DeleteDocument(doc.ID); err != nil {
				t.Logf("Error deleting document %s: %v", doc.ID, err)
			}
		}
	}
}

// createReadmeTestZipFile creates a simple ZIP file with the given content for testing
func createReadmeTestZipFile(filename string, files map[string]string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	for name, content := range files {
		writer, err := zipWriter.Create(name)
		if err != nil {
			return err
		}
		_, err = writer.Write([]byte(content))
		if err != nil {
			return err
		}
	}

	return nil
}

func cleanupReadmeIOTestDocuments(t *testing.T, c *client.Client) {
	testIDs := []string{"first-doc", "second-doc"}
	for _, id := range testIDs {
		resp, err := c.ListDocuments(client.ListOptions{
			Filter:   map[string]interface{}{"external_id": id},
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

	// Also try to clean up by title in case any invalid documents got through
	resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"title": "Invalid Document"},
		PageSize: 1,
	})
	if err != nil {
		t.Logf("Error listing documents for cleanup: %v", err)
		return
	}
	for _, doc := range resp.Documents {
		if err := c.DeleteDocument(doc.ID); err != nil {
			t.Logf("Error deleting document %s: %v", doc.ID, err)
		}
	}
}
