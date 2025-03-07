package integration_test

import (
	"os"
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
	resp, err := c.ListDocuments(map[string]interface{}{"url": "https://example.com/first-post"}, 1)
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
	resp, err = c.ListDocuments(map[string]interface{}{"url": "https://example.com/second-post"}, 1)
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Error("Expected to find second post document")
	}

	// Check post without URL (should be imported)
	resp, err = c.ListDocuments(map[string]interface{}{"title": "Post Without URL"}, 1)
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) == 1 {
		t.Error("Expected to find post without URL")
	}

	// Verify that empty post was not imported
	resp, err = c.ListDocuments(map[string]interface{}{"title": ""}, 1)
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
		resp, err := c.ListDocuments(map[string]interface{}{"url": url}, 1)
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
		resp, err := c.ListDocuments(map[string]interface{}{"title": title}, 1)
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
