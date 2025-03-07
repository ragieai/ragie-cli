package integration_test

import (
	"os"
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
	resp, err := c.ListDocuments(map[string]interface{}{"readmeId": "first-doc"}, 1)
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
	resp, err = c.ListDocuments(map[string]interface{}{"readmeId": "second-doc"}, 1)
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
	resp, err = c.ListDocuments(map[string]interface{}{"title": "Invalid Document"}, 1)
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

func cleanupReadmeIOTestDocuments(t *testing.T, c *client.Client) {
	testIDs := []string{"first-doc", "second-doc"}
	for _, id := range testIDs {
		resp, err := c.ListDocuments(map[string]interface{}{"readmeId": id}, 1)
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
	resp, err := c.ListDocuments(map[string]interface{}{"title": "Invalid Document"}, 1)
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
