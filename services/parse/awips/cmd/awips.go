package main

import (
	"log/slog"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	envFile     string
	logLevelInt int
	logLevel    zerolog.Level = 1
	// The root command of our program
	rootCmd = &cobra.Command{
		Use:   "mds-us-awips",
		Short: "MDS parsing service for US National Weather Service AWIPS products.",
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
	rootCmd.PersistentFlags().IntVar(&logLevelInt, "log", 1, "The logging level to use.")
}

func initConfig() {
	setLogLevel()

	err := godotenv.Load(envFile)
	if err != nil {
		slog.Info("failed to load env file", "error", err.Error())
	}
}

func setLogLevel() {
	logLevel = zerolog.Level(logLevelInt)
}
