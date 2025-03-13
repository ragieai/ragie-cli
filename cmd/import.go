package cmd

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ragie/pkg/client"

	"github.com/beevik/etree"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ImportConfig holds configuration for import operations
type ImportConfig struct {
	DryRun    bool
	Delay     float64
	Partition string
	Mode      string
}

var importCmd = &cobra.Command{
	Use:   "import [type] [file]",
	Short: "Import data from various sources",
	Long: `Import data from various sources into Ragie.

Available import types:

  youtube
    Imports YouTube video transcripts and metadata from a JSON file.
    The JSON file should contain an array of objects with videoId, title, and captions fields.
    Each video will be imported as a separate document with its transcript and metadata.
    Example: ragie import youtube path/to/youtube_videos.json

  wordpress
    Imports WordPress content from an XML export file (WXR format).
    Imports posts, pages, and their metadata including titles, descriptions, and content.
    Each post/page will be imported as a separate document.
    Example: ragie import wordpress path/to/wordpress-export.xml

  readmeio
    Imports ReadmeIO documentation from a ZIP archive.
    The ZIP should contain Markdown files with YAML frontmatter.
    Each Markdown file will be imported as a separate document, preserving metadata.
    Example: ragie import readmeio path/to/readme-docs.zip

  files
    Recursively imports files from a directory.
    All non-empty files will be imported as separate documents.
    Preserves file metadata including path, extension, size, and modification time.
    Example: ragie import files path/to/documents/

  zip
    Imports all files from a zip archive without extracting them.
    Each file will be imported as a separate document.
    Preserves file metadata including path, extension, size, and modification time.
    Example: ragie import zip path/to/documents.zip

Options:
  --mode string    Processing mode: 'hi_res' (high resolution) or 'fast' (default)
                   hi_res: Higher quality processing with better accuracy
                   fast: Faster processing with slightly lower accuracy
                   Note: mode is only supported for 'files' and 'zip' import types`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		importType := args[0]
		file := args[1]

		ragieClient := client.NewClient(viper.GetString("api_key"))
		config := ImportConfig{
			DryRun:    dryRun,
			Delay:     delay,
			Partition: partition,
			Mode:      mode,
		}

		switch importType {
		case "youtube":
			return ImportYouTube(ragieClient, file, config)
		case "wordpress":
			return ImportWordPress(ragieClient, file, config)
		case "readmeio":
			return ImportReadmeIO(ragieClient, file, config)
		case "files":
			return ImportFiles(ragieClient, file, config)
		case "zip":
			return ImportZip(ragieClient, file, config)
		default:
			return fmt.Errorf("unknown import type: %s", importType)
		}
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().StringVar(&mode, "mode", "", "Processing mode: 'hi_res' (high resolution) or 'fast' (default). Only supported for 'files' and 'zip' import types (file upload API).")
}

func documentExists(c *client.Client, config ImportConfig, externalID string) bool {
	opts := client.ListOptions{
		Filter:    map[string]interface{}{"external_id": externalID},
		PageSize:  1,
		Partition: config.Partition,
	}

	resp, err := c.ListDocuments(opts)
	if err != nil {
		return false
	}
	return len(resp.Documents) > 0
}

func createDocumentRaw(c *client.Client, externalID string, name, data string, metadata map[string]interface{}, config ImportConfig) error {
	if config.DryRun {
		fmt.Printf("would save document: %s\n", name)
		return nil
	}

	metadata["external_id"] = externalID

	doc, err := c.CreateDocumentRaw(config.Partition, name, data, metadata)
	if err != nil {
		return err
	}

	fmt.Printf("saved: %s\n", doc.ID)
	return nil
}

// createDocument uploads a file using multipart form data
func createDocument(c *client.Client, externalID string, name string, fileData []byte, fileName string, metadata map[string]interface{}, config ImportConfig) error {
	if config.DryRun {
		fmt.Printf("would save document: %s\n", name)
		return nil
	}

	metadata["external_id"] = externalID

	doc, err := c.CreateDocument(config.Partition, name, fileData, fileName, metadata, config.Mode)
	if err != nil {
		return err
	}

	fmt.Printf("saved: %s\n", doc.ID)
	return nil
}

// ImportYouTube imports YouTube data from a JSON file
func ImportYouTube(c *client.Client, youtubeFile string, config ImportConfig) error {
	fmt.Printf("Loading YouTube JSON file: %s\n", youtubeFile)

	data, err := os.ReadFile(youtubeFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	for _, item := range items {
		videoID, ok := item["videoId"].(string)
		if !ok || videoID == "" {
			fmt.Println("warning: skipping item with no videoId")
			continue
		}

		if documentExists(c, config, videoID) {
			fmt.Printf("warning: skipping video with existing document: %s\n", videoID)
			continue
		}

		title, _ := item["title"].(string)
		captions, _ := item["captions"].([]interface{})

		var content strings.Builder
		if title != "" {
			content.WriteString(title)
			content.WriteString("\n\n")
		}

		for _, cap := range captions {
			if str, ok := cap.(string); ok && str != "" {
				content.WriteString(str)
				content.WriteString("\n")
			}
		}

		if content.Len() == 0 {
			fmt.Printf("warning: refusing to upload empty content: %s\n", videoID)
			continue
		}

		err := createDocumentRaw(c, videoID, title, content.String(), map[string]interface{}{
			"title": title,
		}, config)
		if err != nil {
			fmt.Printf("failed to import video %s: %v\n", videoID, err)
		}

		if config.Delay > 0 {
			time.Sleep(time.Duration(config.Delay * float64(time.Second)))
		}
	}

	return nil
}

// ImportWordPress imports WordPress data from an XML file
func ImportWordPress(c *client.Client, wordpressFile string, config ImportConfig) error {
	fmt.Printf("Loading WordPress XML file: %s\n", wordpressFile)

	doc := etree.NewDocument()
	if err := doc.ReadFromFile(wordpressFile); err != nil {
		return fmt.Errorf("failed to read XML file: %v", err)
	}

	root := doc.Root()
	if root == nil {
		return fmt.Errorf("empty XML file")
	}

	for _, item := range root.FindElements(".//post") {
		metadata := map[string]interface{}{
			"sourceType": "wordpress",
		}

		urlElem := item.FindElement("url")
		url := ""
		if urlElem != nil {
			url = urlElem.Text()
		}
		metadata["url"] = url

		if documentExists(c, config, url) {
			fmt.Printf("warning: skipping post with existing document: %s\n", url)
			continue
		}

		titleElem := item.FindElement("title")
		title := ""
		if titleElem != nil {
			title = titleElem.Text()
		}
		metadata["title"] = title

		descElem := item.FindElement("description")
		desc := ""
		if descElem != nil {
			desc = descElem.Text()
		}

		contentElem := item.FindElement("content")
		content := ""
		if contentElem != nil {
			content = contentElem.Text()
		}

		data := strings.Join([]string{title, desc, content}, "\n\n")

		err := createDocumentRaw(c, url, title, data, metadata, config)
		if err != nil {
			fmt.Printf("failed to import post: %v\n", err)
		}

		if config.Delay > 0 {
			time.Sleep(time.Duration(config.Delay * float64(time.Second)))
		}
	}

	return nil
}

// ImportReadmeIO imports ReadmeIO data from a ZIP file
func ImportReadmeIO(c *client.Client, readmeZip string, config ImportConfig) error {
	fmt.Printf("Loading readme.io ZIP file: %s\n", readmeZip)

	reader, err := zip.OpenReader(readmeZip)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %v", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, ".md") {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			fmt.Printf("failed to open file in zip %s: %v\n", file.Name, err)
			continue
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			fmt.Printf("failed to read file in zip %s: %v\n", file.Name, err)
			continue
		}

		contentStr := string(content)
		if strings.TrimSpace(contentStr) == "" {
			fmt.Printf("warning: refusing to upload empty content: %s\n", file.Name)
			continue
		}

		metadata := map[string]interface{}{
			"sourceType": "readmeio",
		}

		// Parse frontmatter
		parts := strings.SplitN(contentStr, "---", 3)
		if len(parts) >= 3 {
			frontmatter := parts[1]
			contentStr = parts[2]

			for _, line := range strings.Split(frontmatter, "\n") {
				if strings.Contains(line, ":") {
					parts := strings.SplitN(line, ":", 2)
					key := strings.TrimSpace(parts[0])
					value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
					metadata[key] = value
				}
			}
		}

		docID, _ := metadata["slug"].(string)
		if docID == "" {
			fmt.Printf("warning: skipping document without slug: %s\n", file.Name)
			continue
		}

		metadata["readmeId"] = docID

		if documentExists(c, config, docID) {
			fmt.Printf("warning: skipping document with existing id: %s\n", docID)
			continue
		}

		title, _ := metadata["title"].(string)
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(file.Name), ".md")
		}

		err = createDocumentRaw(c, docID, title, contentStr, metadata, config)
		if err != nil {
			fmt.Printf("failed to import readme document %s: %v\n", file.Name, err)
		}

		if config.Delay > 0 {
			time.Sleep(time.Duration(config.Delay * float64(time.Second)))
		}
	}

	return nil
}

// ImportFiles imports all files from a directory recursively
func ImportFiles(c *client.Client, directory string, config ImportConfig) error {
	fmt.Printf("Loading files from directory: %s\n", directory)

	// Check if directory exists
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("failed to access directory: %v", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", directory)
	}

	// Walk through the directory recursively
	return filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("error accessing path %s: %v\n", path, err)
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Generate a unique external ID based on the relative path
		relPath, err := filepath.Rel(directory, path)
		if err != nil {
			fmt.Printf("error getting relative path for %s: %v\n", path, err)
			return nil
		}
		externalID := filepath.ToSlash(relPath)

		// Skip if document already exists
		if documentExists(c, config, externalID) {
			fmt.Printf("warning: skipping file with existing document: %s\n", externalID)
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("error reading file %s: %v\n", path, err)
			return nil
		}

		// Skip empty files
		if len(strings.TrimSpace(string(content))) == 0 {
			fmt.Printf("warning: skipping empty file: %s\n", path)
			return nil
		}

		metadata := map[string]interface{}{
			"source_type": "files",
			"path":        externalID,
			"extension":   filepath.Ext(path),
			"size":        info.Size(),
			"mod_time":    info.ModTime().Format(time.RFC3339),
		}

		err = createDocument(c, externalID, filepath.Base(path), content, filepath.Base(path), metadata, config)
		if err != nil {
			fmt.Printf("failed to import file %s: %v\n", path, err)
		}

		if config.Delay > 0 {
			time.Sleep(time.Duration(config.Delay * float64(time.Second)))
		}

		return nil
	})
}

// ImportZip imports all files from a zip archive without extracting them
func ImportZip(c *client.Client, zipFile string, config ImportConfig) error {
	fmt.Printf("Loading files from zip archive: %s\n", zipFile)

	// Open the zip file
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %v", err)
	}
	defer reader.Close()

	// Process each file in the zip
	for _, file := range reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Generate a unique external ID based on the path within the zip
		externalID := filepath.ToSlash(file.Name)

		// Skip if document already exists
		if documentExists(c, config, externalID) {
			fmt.Printf("warning: skipping file with existing document: %s\n", externalID)
			continue
		}

		// Open the file within the zip
		rc, err := file.Open()
		if err != nil {
			fmt.Printf("failed to open file in zip %s: %v\n", file.Name, err)
			continue
		}

		// Read file content
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			fmt.Printf("failed to read file in zip %s: %v\n", file.Name, err)
			continue
		}

		// Skip empty files
		if len(strings.TrimSpace(string(content))) == 0 {
			fmt.Printf("warning: skipping empty file: %s\n", file.Name)
			continue
		}

		// Create metadata for the file
		metadata := map[string]interface{}{
			"source_type":     "zip",
			"path":            externalID,
			"extension":       filepath.Ext(file.Name),
			"size":            file.UncompressedSize64,
			"mod_time":        file.Modified.Format(time.RFC3339),
			"compressed_size": file.CompressedSize64,
			"zip_source":      filepath.Base(zipFile),
		}

		// Create the document using multipart form data
		err = createDocument(c, externalID, filepath.Base(file.Name), content, file.Name, metadata, config)
		if err != nil {
			fmt.Printf("failed to import file %s: %v\n", file.Name, err)
		}

		if config.Delay > 0 {
			time.Sleep(time.Duration(config.Delay * float64(time.Second)))
		}
	}

	return nil
}
