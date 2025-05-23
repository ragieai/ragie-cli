package cmd

import (
	"fmt"
	"testing"

	"ragie/pkg/client"

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
		{
			name:        "valid command with audio flag for files",
			args:        []string{"--audio", "files", "/tmp/test"},
			expectError: false,
		},
		{
			name:        "invalid command with audio flag for youtube",
			args:        []string{"--audio", "youtube", "/tmp/test"},
			expectError: true,
			errorMsg:    "--audio flag is only supported for 'files' and 'zip' import types",
		},
		{
			name:        "valid command with video flag for files",
			args:        []string{"--video=audio_only", "files", "/tmp/test"},
			expectError: false,
		},
		{
			name:        "valid command with video flag audio_video for zip",
			args:        []string{"--video=audio_video", "zip", "/tmp/test.zip"},
			expectError: false,
		},
		{
			name:        "valid command with video flag video_only for files",
			args:        []string{"--video=video_only", "files", "/tmp/test"},
			expectError: false,
		},
		{
			name:        "invalid command with video flag for youtube",
			args:        []string{"--video=audio_only", "youtube", "/tmp/test"},
			expectError: true,
			errorMsg:    "--video flag is only supported for 'files' and 'zip' import types",
		},
		{
			name:        "invalid video flag value",
			args:        []string{"--video=invalid", "files", "/tmp/test"},
			expectError: true,
			errorMsg:    "--video must be either 'audio_only', 'video_only', or 'audio_video'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags for each test
			force = false
			replace = false
			audio = false
			video = ""

			// Create a new command for testing
			cmd := &cobra.Command{
				Use:  "import [type] [file]",
				Args: cobra.ExactArgs(2),
				RunE: func(cmd *cobra.Command, args []string) error {
					importType := args[0]

					// Validate that --force and --replace are mutually exclusive
					if force && replace {
						return fmt.Errorf("--force and --replace flags cannot be used together")
					}

					// Validate that audio is only used with files and zip import types
					if audio && importType != "files" && importType != "zip" {
						return fmt.Errorf("--audio flag is only supported for 'files' and 'zip' import types")
					}

					// Validate video flag values
					if video != "" && video != "audio_only" && video != "video_only" && video != "audio_video" {
						return fmt.Errorf("--video must be either 'audio_only', 'video_only', or 'audio_video'")
					}

					// Validate that video is only used with files and zip import types
					if video != "" && importType != "files" && importType != "zip" {
						return fmt.Errorf("--video flag is only supported for 'files' and 'zip' import types")
					}

					return nil
				},
			}

			cmd.Flags().BoolVar(&force, "force", false, "Force import")
			cmd.Flags().BoolVar(&replace, "replace", false, "Replace existing documents")
			cmd.Flags().BoolVar(&audio, "audio", false, "Enable audio processing")
			cmd.Flags().StringVar(&video, "video", "", "Video processing mode")

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

func TestConstructMode(t *testing.T) {
	tests := []struct {
		name     string
		config   ImportConfig
		expected interface{}
	}{
		{
			name:     "no flags",
			config:   ImportConfig{},
			expected: nil,
		},
		{
			name:     "mode only",
			config:   ImportConfig{Mode: "fast"},
			expected: "fast",
		},
		{
			name:     "audio only",
			config:   ImportConfig{Audio: true},
			expected: &client.Mode{Audio: true},
		},
		{
			name:     "static only",
			config:   ImportConfig{Static: "hi_res"},
			expected: &client.Mode{Static: "hi_res"},
		},
		{
			name:     "mode fast + audio",
			config:   ImportConfig{Mode: "fast", Audio: true},
			expected: &client.Mode{Audio: true},
		},
		{
			name:     "mode fast + static + audio",
			config:   ImportConfig{Mode: "fast", Static: "hi_res", Audio: true},
			expected: &client.Mode{Static: "hi_res", Audio: true},
		},
		{
			name:     "mode all + static",
			config:   ImportConfig{Mode: "all", Static: "fast"},
			expected: &client.Mode{Static: "fast", Audio: true, Video: "audio_video"},
		},
		{
			name:     "mode all + static + audio (audio should still be true)",
			config:   ImportConfig{Mode: "all", Static: "fast", Audio: true},
			expected: &client.Mode{Static: "fast", Audio: true, Video: "audio_video"},
		},
		{
			name:     "video only",
			config:   ImportConfig{Video: "audio_only"},
			expected: &client.Mode{Video: "audio_only"},
		},
		{
			name:     "video video_only",
			config:   ImportConfig{Video: "video_only"},
			expected: &client.Mode{Video: "video_only"},
		},
		{
			name:     "video audio_video",
			config:   ImportConfig{Video: "audio_video"},
			expected: &client.Mode{Video: "audio_video"},
		},
		{
			name:     "mode fast + video",
			config:   ImportConfig{Mode: "fast", Video: "audio_only"},
			expected: &client.Mode{Video: "audio_only"},
		},
		{
			name:     "mode all + video (explicit video overrides)",
			config:   ImportConfig{Mode: "all", Video: "video_only"},
			expected: &client.Mode{Audio: true, Video: "video_only"},
		},
		{
			name:     "static + audio + video",
			config:   ImportConfig{Static: "hi_res", Audio: true, Video: "audio_video"},
			expected: &client.Mode{Static: "hi_res", Audio: true, Video: "audio_video"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConstructMode(tt.config)

			// Compare results
			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			switch expectedMode := tt.expected.(type) {
			case string:
				if result != expectedMode {
					t.Errorf("Expected %q, got %v", expectedMode, result)
				}
			case *client.Mode:
				resultMode, ok := result.(*client.Mode)
				if !ok {
					t.Errorf("Expected *client.Mode, got %T", result)
					return
				}
				if resultMode.Static != expectedMode.Static {
					t.Errorf("Expected Static %q, got %q", expectedMode.Static, resultMode.Static)
				}
				if resultMode.Audio != expectedMode.Audio {
					t.Errorf("Expected Audio %v, got %v", expectedMode.Audio, resultMode.Audio)
				}
				if resultMode.Video != expectedMode.Video {
					t.Errorf("Expected Video %q, got %q", expectedMode.Video, resultMode.Video)
				}
			default:
				t.Errorf("Unexpected expected type: %T", tt.expected)
			}
		})
	}
}
