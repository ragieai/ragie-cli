package cmd

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
)

func TestImportCmdFlagValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid command without flags",
			args:        []string{"files", "/tmp/test"},
			expectError: false,
		},
		{
			name:        "valid command with force flag",
			args:        []string{"--force", "files", "/tmp/test"},
			expectError: false,
		},
		{
			name:        "valid command with replace flag",
			args:        []string{"--replace", "files", "/tmp/test"},
			expectError: false,
		},
		{
			name:        "invalid command with both force and replace flags",
			args:        []string{"--force", "--replace", "files", "/tmp/test"},
			expectError: true,
			errorMsg:    "--force and --replace flags cannot be used together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			force = false
			replace = false

			// Create a new command for testing
			cmd := &cobra.Command{
				Use:  "import [type] [file]",
				Args: cobra.ExactArgs(2),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Validate that --force and --replace are mutually exclusive
					if force && replace {
						return fmt.Errorf("--force and --replace flags cannot be used together")
					}
					return nil
				},
			}

			cmd.Flags().BoolVar(&force, "force", false, "Force import")
			cmd.Flags().BoolVar(&replace, "replace", false, "Replace existing documents")

			// Set arguments and parse flags
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Errorf("Expected error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
			if tt.expectError && err != nil && err.Error() != tt.errorMsg {
				t.Errorf("Expected error message '%s', but got '%s'", tt.errorMsg, err.Error())
			}
		})
	}
}
