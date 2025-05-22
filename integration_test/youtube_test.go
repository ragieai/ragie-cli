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
	resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": "test123"},
		PageSize: 1,
	})
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
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": "test456"},
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Error("Expected to find test456 video document")
	}

	// Verify that invalid video was not imported
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"title": "Invalid Video"},
		PageSize: 1,
	})
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

func TestYouTubeImportForce(t *testing.T) {
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

	testVideoID := "force_test_video"

	// Clean up any existing test documents
	cleanupForceTestDocuments(t, c, []string{testVideoID})

	// Create temporary test file
	tempFile := filepath.Join(t.TempDir(), "youtube_force_test.json")
	testData := `[
		{
			"videoId": "` + testVideoID + `",
			"title": "Force Test Video",
			"captions": ["This is a test video for force flag"]
		}
	]`

	if err := os.WriteFile(tempFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// First import without force - should succeed
	t.Log("Running first YouTube import...")
	config := cmd.ImportConfig{
		DryRun: false,
		Delay:  0,
		Force:  false,
	}

	err := cmd.ImportYouTube(c, tempFile, config)
	if err != nil {
		t.Fatalf("Failed to import YouTube data: %v", err)
	}

	time.Sleep(1 * time.Second) // Give API time to process

	// Verify document was created
	resp, err := c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testVideoID},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(resp.Documents))
	}
	firstDocID := resp.Documents[0].ID

	// Second import without force - should skip
	t.Log("Running second YouTube import without force...")
	err = cmd.ImportYouTube(c, tempFile, config)
	if err != nil {
		t.Fatalf("Failed to import YouTube data: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify still only one document
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testVideoID},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 1 {
		t.Errorf("Expected 1 document after second import without force, got %d", len(resp.Documents))
	}

	// Third import with force - should create duplicate
	t.Log("Running third YouTube import with force...")
	config.Force = true
	err = cmd.ImportYouTube(c, tempFile, config)
	if err != nil {
		t.Fatalf("Failed to import YouTube data with force: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Verify now two documents exist
	resp, err = c.ListDocuments(client.ListOptions{
		Filter:   map[string]interface{}{"external_id": testVideoID},
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(resp.Documents) != 2 {
		t.Errorf("Expected 2 documents after force import, got %d", len(resp.Documents))
	}

	// Verify they have different IDs
	if len(resp.Documents) == 2 {
		if resp.Documents[0].ID == resp.Documents[1].ID {
			t.Error("Expected different document IDs, but they are the same")
		}
		// Verify the first document is still there
		foundFirst := false
		for _, doc := range resp.Documents {
			if doc.ID == firstDocID {
				foundFirst = true
				break
			}
		}
		if !foundFirst {
			t.Error("Original document was not preserved during force import")
		}
	}

	// Clean up test documents
	cleanupForceTestDocuments(t, c, []string{testVideoID})
}

func cleanupTestDocuments(t *testing.T, c *client.Client) {
	testIDs := []string{"test123", "test456", "test789"}
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
}

// cleanupForceTestDocuments cleans up test documents by external ID (supports multiple documents with same external_id)
func cleanupForceTestDocuments(t *testing.T, c *client.Client, externalIDs []string) {
	for _, id := range externalIDs {
		resp, err := c.ListDocuments(client.ListOptions{
			Filter:   map[string]interface{}{"external_id": id},
			PageSize: 100, // Get all documents with this external_id
		})
		if err != nil {
			t.Logf("Error listing documents for cleanup: %v", err)
			continue
		}
		for _, doc := range resp.Documents {
			if err := c.DeleteDocument(doc.ID); err != nil {
				t.Logf("Error deleting document %s: %v", doc.ID, err)
			} else {
				t.Logf("Cleaned up document: %s", doc.ID)
			}
		}
	}
}
