package main

import (
	"log/slog"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	envFile string
	// The root command of our program
	rootCmd = &cobra.Command{
		Use:   "mds-us-ingest",
		Short: "MDS ingest services for the United States.",
		Long: `The Meteorological Data System (MDS) uses several modules of services to make up the global system.
		These services assist in collecting, processing, and serving United States data.`,
	}
)

// Go, go, go
func main() {
	rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Bind our args to the command
	rootCmd.PersistentFlags().StringVar(&envFile, "env", ".env", "The env file to read.")

	rootCmd.AddCommand(nwwsCmd)
}

func initConfig() {
	err := godotenv.Load(envFile)
	if err != nil {
		slog.Info("failed to load env file", "error", err.Error())
	}
}
