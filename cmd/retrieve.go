package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"ragie/pkg/client"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	topK                 int
	filter               string
	rerank               bool
	maxChunksPerDocument int
	recencyBias          bool
)

var retrieveCmd = &cobra.Command{
	Use:   "retrieve [query]",
	Short: "Retrieve documents from Ragie",
	Long: `Retrieve documents from Ragie using semantic search.
The query will be used to find the most relevant documents in your knowledge base.
Results are returned as JSON with document content, metadata, and relevance scores.

Available options:
- --top-k: Maximum number of results to return (default: 8)
- --filter: JSON filter to apply to the search
- --rerank: Rerank chunks for semantic relevancy post cosine similarity
- --max-chunks-per-document: Maximum number of chunks to retrieve per document
- --recency-bias: Enable recency bias to favor more recent documents
- --partition: The partition to scope the retrieval to`,
	Args: cobra.ExactArgs(1),
	RunE: runRetrieve,
}

func init() {
	rootCmd.AddCommand(retrieveCmd)

	retrieveCmd.Flags().IntVar(&topK, "top-k", 0, "Maximum number of results to return")
	retrieveCmd.Flags().StringVar(&filter, "filter", "", "JSON filter to apply to the search (e.g., '{\"source_type\":\"files\"}')")
	retrieveCmd.Flags().BoolVar(&rerank, "rerank", false, "Rerank chunks for semantic relevancy post cosine similarity")
	retrieveCmd.Flags().IntVar(&maxChunksPerDocument, "max-chunks-per-document", 0, "Maximum number of chunks to retrieve per document")
	retrieveCmd.Flags().BoolVar(&recencyBias, "recency-bias", false, "Enable recency bias to favor more recent documents")
}

func runRetrieve(cmd *cobra.Command, args []string) error {
	query := args[0]

	// Parse filter if provided
	var filterMap map[string]interface{}
	if filter != "" {
		if err := json.Unmarshal([]byte(filter), &filterMap); err != nil {
			return fmt.Errorf("invalid filter JSON: %w", err)
		}
	}

	ragieApiKey := viper.GetString("api_key")
	if ragieApiKey == "" {
		return fmt.Errorf("RAGIE_API_KEY environment variable must be set. Please set it with:\nexport RAGIE_API_KEY=your_ragie_api_key")
	}

	ragieClient := client.NewClient(ragieApiKey)

	opts := client.RetrievalOptions{
		Query:  query,
		Filter: filterMap,
	}

	// Only set fields if they were explicitly provided via flags
	if cmd.Flags().Changed("top-k") {
		opts.TopK = &topK
	}
	if cmd.Flags().Changed("partition") {
		opts.Partition = &partition
	}
	if cmd.Flags().Changed("rerank") {
		opts.Rerank = &rerank
	}
	if cmd.Flags().Changed("max-chunks-per-document") {
		opts.MaxChunksPerDocument = &maxChunksPerDocument
	}
	if cmd.Flags().Changed("recency-bias") {
		opts.RecencyBias = &recencyBias
	}

	response, err := ragieClient.Retrieve(opts)
	if err != nil {
		return fmt.Errorf("retrieval failed: %w", err)
	}

	// Output results as JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return nil
}
