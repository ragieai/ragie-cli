package integration_test

import (
	"os"
	"testing"
	"time"

	"ragie-cli/cmd"
	"ragie-cli/pkg/client"

	"github.com/spf13/viper"
)

func TestYouTubeImport(t *testing.T) {
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
	cleanupTestDocuments(t, c)

	// Run the import
	t.Log("Running YouTube import...")
	config := cmd.ImportConfig{
		DryRun: false,
		Delay:  0, // No delay for tests
	}
	err := cmd.ImportYouTube(c, "../testdata/youtube_sample.json", config)
	if err != nil {
		t.Fatalf("Failed to import YouTube data: %v", err)
	}

	// Verify the imports
	t.Log("Verifying imported documents...")
	time.Sleep(1 * time.Second) // Give API some time to process

	// Check first video
	resp, err := c.ListDocuments(map[string]interface{}{"videoId": "test123"}, 1)
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Error("Expected to find test123 video document")
	} else {
		doc := resp.Documents[0]
		if doc.Name != "Test Video 1" {
			t.Errorf("Expected title 'Test Video 1', got '%s'", doc.Name)
		}
	}

	// Check second video
	resp, err = c.ListDocuments(map[string]interface{}{"videoId": "test456"}, 1)
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Error("Expected to find test456 video document")
	}

	// Verify that invalid video was not imported
	resp, err = c.ListDocuments(map[string]interface{}{"title": "Invalid Video"}, 1)
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 0 {
		t.Error("Invalid video should not have been imported")
	}

	// Clean up test documents
	t.Log("Cleaning up test documents...")
	cleanupTestDocuments(t, c)
}

func cleanupTestDocuments(t *testing.T, c *client.Client) {
	testIDs := []string{"test123", "test456", "test789"}
	for _, id := range testIDs {
		resp, err := c.ListDocuments(map[string]interface{}{"videoId": id}, 1)
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
