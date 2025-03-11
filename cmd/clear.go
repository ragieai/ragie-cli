package cmd

import (
	"fmt"

	"ragie/pkg/client"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all documents",
	Long: `Clear all documents from Ragie.
If a partition is specified, only documents in that partition will be cleared.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Running clear...")

		c := client.NewClient(viper.GetString("api_key"))
		opts := client.ListOptions{
			Filter:    map[string]interface{}{},
			PageSize:  100,
			Partition: partition,
		}

		for {
			resp, err := c.ListDocuments(opts)
			if err != nil {
				return fmt.Errorf("failed to list documents: %v", err)
			}

			if len(resp.Documents) == 0 {
				break
			}

			for _, doc := range resp.Documents {
				if dryRun {
					fmt.Printf("would delete %s\n", doc.ID)
					continue
				}

				if err := c.DeleteDocument(doc.ID); err != nil {
					fmt.Printf("error deleting document: %v\n", err)
					continue
				}

				fmt.Printf("deleted %s\n", doc.ID)
			}

			if resp.Pagination.NextCursor == "" {
				break
			}
			opts.Cursor = resp.Pagination.NextCursor
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(clearCmd)
}
