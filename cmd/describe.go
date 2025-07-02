package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"ragie/pkg/client"

	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	shellEscape bool
	maxSamples  int
)

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Generate a description for a Ragie retrieval tool",
	Long: `Generate a description for a Ragie retrieval tool by analyzing document summaries
from the specified partition. This command uses OpenAI to create a coherent description
of the knowledge base contents.`,
	RunE: runDescribe,
}

func init() {
	rootCmd.AddCommand(describeCmd)

	describeCmd.Flags().BoolVar(&shellEscape, "shell-escape", false, "Output description in JSON and shell-safe format")
	describeCmd.Flags().IntVar(&maxSamples, "max-samples", 10, "Maximum number of documents to use")
}

func runDescribe(cmd *cobra.Command, args []string) error {
	// Check for required environment variables
	openaiApiKey := os.Getenv("OPENAI_API_KEY")
	if openaiApiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable must be set. Please set it with:\nexport OPENAI_API_KEY=your_openai_api_key")
	}

	ragieApiKey := viper.GetString("api_key")
	if ragieApiKey == "" {
		return fmt.Errorf("RAGIE_API_KEY environment variable must be set. Please set it with:\nexport RAGIE_API_KEY=your_ragie_api_key")
	}

	// Initialize clients
	ragieClient := client.NewClient(ragieApiKey)
	openaiClient := openai.NewClient(openaiApiKey)

	// Get partition from flags
	partition := viper.GetString("partition")

	// Generate description
	description, err := generateDescription(ragieClient, openaiClient, partition, maxSamples)
	if err != nil {
		return fmt.Errorf("failed to generate description: %w", err)
	}

	// Output the result
	fmt.Println("\nDescription:\n")

	if shellEscape {
		escapedDescription := strings.ReplaceAll(description, `"`, `\"`)
		fmt.Println(escapedDescription)
	} else {
		fmt.Println(description)
	}

	return nil
}

func generateDescription(ragieClient *client.Client, openaiClient *openai.Client, partition string, maxSamples int) (string, error) {
	summaries, err := getSamples(ragieClient, partition, maxSamples)
	if err != nil {
		return "", fmt.Errorf("failed to get samples: %w", err)
	}

	if len(summaries) == 0 {
		return "", fmt.Errorf("no document summaries found in partition: %s", partition)
	}

	// Collapse summaries
	collapsed := summaries[0]
	for i := 1; i < len(summaries); i++ {
		fmt.Println("collapsing...")
		var err error
		collapsed, err = collapse(openaiClient, collapsed, summaries[i])
		if err != nil {
			return "", fmt.Errorf("failed to collapse summaries: %w", err)
		}
	}

	// Rephrase the final result
	rephrased, err := rephrase(openaiClient, collapsed)
	if err != nil {
		return "", fmt.Errorf("failed to rephrase description: %w", err)
	}

	return rephrased, nil
}

func getSamples(ragieClient *client.Client, partition string, maxSamples int) ([]string, error) {
	var summaries []string
	count := 0

	// Get documents
	resp, err := ragieClient.ListDocuments(client.ListOptions{
		Partition: partition,
		PageSize:  maxSamples,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	// Get summaries for each document
	for _, doc := range resp.Documents {
		summary, err := ragieClient.GetDocumentSummary(doc.ID, partition)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: No summary for document %s: %v\n", doc.ID, err)
			continue
		}

		summaries = append(summaries, summary.Summary)
		count++

		if count >= maxSamples {
			break
		}
	}

	return summaries, nil
}

func collapse(openaiClient *openai.Client, existing, summary string) (string, error) {
	prompt := fmt.Sprintf(`Combine the following summaries into a single, coherent tool description. The tool is a knowledge base retrieval tool.
The generated description should be a single paragraph of no more than 300 words.

<existing_description>
%s
</existing_description>

<new_summary>
%s
</new_summary>`, existing, summary)

	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "gpt-4o-mini",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("no content generated from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}

func rephrase(openaiClient *openai.Client, text string) (string, error) {
	prompt := fmt.Sprintf(`The following text is a summary of the contents in a knowledge base.

Describe the contents of the knowledge base in a way that is useful for an LLM to route tool calls to this "retrieve" tool.

The generated description should be no more than 300 words.

<text>
%s
</text>`, text)

	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "gpt-4o",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("no content generated from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}
