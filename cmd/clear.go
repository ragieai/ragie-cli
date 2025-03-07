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
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Running clear...")

		c := client.NewClient(viper.GetString("api_key"))

		for {
			resp, err := c.ListDocuments("", map[string]interface{}{}, 0)
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
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(clearCmd)
}
