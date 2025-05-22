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

func TestWordPressImport(t *testing.T) {
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
	cleanupWordPressTestDocuments(t, c)

	// Run the import
	t.Log("Running WordPress import...")
	config := cmd.ImportConfig{
		DryRun: false,
		Delay:  0, // No delay for tests
	}
	err := cmd.ImportWordPress(c, "../testdata/wordpress_sample.xml", config)
	if err != nil {
		t.Fatalf("Failed to import WordPress data: %v", err)
	}

	// Verify the imports
	t.Log("Verifying imported documents...")
	time.Sleep(1 * time.Second) // Give API some time to process

	// Check first post
	resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": "https://example.com/first-post"},
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Error("Expected to find first post document")
	} else {
		doc := resp.Documents[0]
		if doc.Name != "First Test Post" {
			t.Errorf("Expected title 'First Test Post', got '%s'", doc.Name)
		}
		if doc.Metadata["sourceType"] != "wordpress" {
			t.Errorf("Expected sourceType 'wordpress', got '%v'", doc.Metadata["sourceType"])
		}
	}

	// Check second post
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": "https://example.com/second-post"},
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Error("Expected to find second post document")
	}

	// Check post without URL (should be imported)
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"title": "Post Without URL"},
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) == 1 {
		t.Error("Expected to find post without URL")
	}

	// Verify that empty post was not imported
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"title": ""},
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) == 0 {
		t.Error("Empty post should not have been imported")
	}

	// Clean up test documents
	t.Log("Cleaning up test documents...")
	cleanupWordPressTestDocuments(t, c)
}

func TestWordPressImportForce(t *testing.T) {
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

	testURL := "https://example.com/force-test-post"

	// Clean up any existing test documents with this external ID
	if resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testURL},
		PageSize: 100,
	}); err == nil {
		for _, doc := range resp.Documents {
			c.DeleteDocument(doc.ID)
		}
	}

	// Create temporary WordPress XML file
	tempFile := filepath.Join(t.TempDir(), "wordpress_force_test.xml")
	testXML := `<?xml version="1.0" encoding="UTF-8"?>
<root>
	<post>
		<url>` + testURL + `</url>
		<title>Force Test Post</title>
		<description>Test description</description>
		<content>This is test content for force flag testing</content>
	</post>
</root>`

	if err := os.WriteFile(tempFile, []byte(testXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// First import without force
	t.Log("Running first WordPress import...")
	config := cmd.ImportConfig{
		DryRun: false,
		Delay:  0,
		Force:  false,
	}

	err := cmd.ImportWordPress(c, tempFile, config)
	if err != nil {
		t.Fatalf("Failed to import WordPress data: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify document was created
	resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testURL},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(resp.Documents))
	}

	// Second import without force - should skip
	t.Log("Running second WordPress import without force...")
	err = cmd.ImportWordPress(c, tempFile, config)
	if err != nil {
		t.Fatalf("Failed to import WordPress data: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify still only one document
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testURL},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Errorf("Expected 1 document after second import without force, got %d", len(resp.Documents))
	}

	// Third import with force - should create duplicate
	t.Log("Running third WordPress import with force...")
	config.Force = true
	err = cmd.ImportWordPress(c, tempFile, config)
	if err != nil {
		t.Fatalf("Failed to import WordPress data with force: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify now two documents exist
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testURL},
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
		Filter:   map[string]interface{}{"external_id": testURL},
		PageSize: 100,
	}); err == nil {
		for _, doc := range resp.Documents {
			if err := c.DeleteDocument(doc.ID); err != nil {
				t.Logf("Error deleting document %s: %v", doc.ID, err)
			}
		}
	}
}

func TestWordPressImportReplace(t *testing.T) {
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

	testURL := "https://example.com/replace-test-post"

	// Clean up any existing test documents with this external ID
	if resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testURL},
		PageSize: 100,
	}); err == nil {
		for _, doc := range resp.Documents {
			c.DeleteDocument(doc.ID)
		}
	}

	// Create temporary WordPress XML file with first version
	tempFile := filepath.Join(t.TempDir(), "wordpress_replace_test.xml")
	testXML1 := `<?xml version="1.0" encoding="UTF-8"?>
<root>
	<post>
		<url>` + testURL + `</url>
		<title>Replace Test Post - Version 1</title>
		<description>Test description - Version 1</description>
		<content>This is the first version of the test content</content>
	</post>
</root>`

	if err := os.WriteFile(tempFile, []byte(testXML1), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// First import - create initial document
	t.Log("Running first WordPress import...")
	config := cmd.ImportConfig{
		DryRun:  false,
		Delay:   0,
		Force:   false,
		Replace: false,
	}

	err := cmd.ImportWordPress(c, tempFile, config)
	if err != nil {
		t.Fatalf("Failed to import WordPress data: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify document was created
	resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testURL},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(resp.Documents))
	}
	firstDocID := resp.Documents[0].ID

	// Update XML file with second version
	testXML2 := `<?xml version="1.0" encoding="UTF-8"?>
<root>
	<post>
		<url>` + testURL + `</url>
		<title>Replace Test Post - Version 2</title>
		<description>Test description - Version 2</description>
		<content>This is the second version of the test content</content>
	</post>
</root>`

	if err := os.WriteFile(tempFile, []byte(testXML2), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// Second import with replace - should replace the existing document
	t.Log("Running second WordPress import with replace...")
	config.Replace = true
	err = cmd.ImportWordPress(c, tempFile, config)
	if err != nil {
		t.Fatalf("Failed to import WordPress data with replace: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify still only one document but with different ID (replaced)
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testURL},
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
		Filter:   map[string]interface{}{"external_id": testURL},
		PageSize: 100,
	}); err == nil {
		for _, doc := range resp.Documents {
			if err := c.DeleteDocument(doc.ID); err != nil {
				t.Logf("Error deleting document %s: %v", doc.ID, err)
			}
		}
	}
}

func cleanupWordPressTestDocuments(t *testing.T, c *client.Client) {
	testURLs := []string{
		"https://example.com/first-post",
		"https://example.com/second-post",
		"",
	}
	testTitles := []string{
		"Post Without URL",
	}

	// Clean up by URL
	for _, url := range testURLs {
		resp, err := c.ListDocuments(client.ListOptions{
			Filter:   map[string]interface{}{"external_id": url},
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

	// Clean up by title (for posts without URL)
	for _, title := range testTitles {
		resp, err := c.ListDocuments(client.ListOptions{
			Filter:   map[string]interface{}{"title": title},
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
