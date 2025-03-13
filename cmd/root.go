package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dryRun    bool
	delay     float64
	partition string
	mode      string
)

var rootCmd = &cobra.Command{
	Use:   filepath.Base(os.Args[0]),
	Short: "A CLI tool for importing data into Ragie",
	Long: `A command line interface for importing various data formats into Ragie,
including YouTube data, WordPress exports, and ReadmeIO documentation.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Print what would happen without making changes")
	rootCmd.PersistentFlags().Float64Var(&delay, "delay", 2.0, "Delay between imports in seconds")
	rootCmd.PersistentFlags().StringVar(&partition, "partition", "", "Optional partition to use for operations")
}

func initConfig() {
	apiKey := os.Getenv("RAGIE_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: RAGIE_API_KEY environment variable must be set")
		os.Exit(1)
	}
	viper.Set("api_key", apiKey)
}
