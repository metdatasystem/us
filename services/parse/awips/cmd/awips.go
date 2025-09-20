package main

import (
	"log/slog"

	"github.com/joho/godotenv"
	awips "github.com/metdatasystem/us/services/parse/awips/internal"
	"github.com/spf13/cobra"
)

var (
	envFile string
	// The root command of our program
	rootCmd = &cobra.Command{
		Use:   "mds-us-awips",
		Short: "MDS parsing service for US National Weather Service AWIPS products.",
		Run: func(cmd *cobra.Command, args []string) {
			awips.Go()
		},
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
}

func initConfig() {
	err := godotenv.Load(envFile)
	if err != nil {
		slog.Info("failed to load env file", "error", err.Error())
	}
}
