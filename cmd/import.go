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

	"ragie-cli/pkg/client"

	"github.com/beevik/etree"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ImportConfig holds configuration for import operations
type ImportConfig struct {
	DryRun bool
	Delay  float64
}

var importCmd = &cobra.Command{
	Use:   "import [type] [file]",
	Short: "Import data from various sources",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		importType := args[0]
		file := args[1]

		ragieClient := client.NewClient(viper.GetString("api_key"))
		config := ImportConfig{
			DryRun: dryRun,
			Delay:  delay,
		}

		switch importType {
		case "youtube":
			return ImportYouTube(ragieClient, file, config)
		case "wordpress":
			return ImportWordPress(ragieClient, file, config)
		case "readmeio":
			return ImportReadmeIO(ragieClient, file, config)
		default:
			return fmt.Errorf("unknown import type: %s", importType)
		}
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}

func documentExists(c *client.Client, filter map[string]interface{}) bool {
	resp, err := c.ListDocuments(filter, 1)
	if err != nil {
		return false
	}
	return len(resp.Documents) > 0
}

func createDocumentRaw(c *client.Client, name, data string, metadata map[string]interface{}, dryRun bool) error {
	if dryRun {
		fmt.Printf("would save document: %s\n", name)
		return nil
	}

	doc, err := c.CreateDocumentRaw(name, data, metadata)
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

		if documentExists(c, map[string]interface{}{"videoId": videoID}) {
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

		err := createDocumentRaw(c, title, content.String(), map[string]interface{}{
			"title":   title,
			"videoId": videoID,
		}, config.DryRun)
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

		if documentExists(c, map[string]interface{}{"url": url}) {
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

		err := createDocumentRaw(c, title, data, metadata, config.DryRun)
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

		if documentExists(c, map[string]interface{}{"readmeId": docID}) {
			fmt.Printf("warning: skipping document with existing id: %s\n", docID)
			continue
		}

		title, _ := metadata["title"].(string)
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(file.Name), ".md")
		}

		err = createDocumentRaw(c, title, contentStr, metadata, config.DryRun)
		if err != nil {
			fmt.Printf("failed to import readme document %s: %v\n", file.Name, err)
		}

		if config.Delay > 0 {
			time.Sleep(time.Duration(config.Delay * float64(time.Second)))
		}
	}

	return nil
}
