package main

import (
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	envFile     string
	logLevelInt int
	logLevel    zerolog.Level = 1
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
	rootCmd.PersistentFlags().IntVar(&logLevelInt, "log", 1, "The logging level to use.")

	rootCmd.AddCommand(nwwsCmd)
	rootCmd.AddCommand(localCmd)
}

func initConfig() {
	setLogLevel()

	err := godotenv.Load(envFile)
	if err != nil {
		log.Error().Err(err).Msg("failed to load env file")
	}
}

func setLogLevel() {
	logLevel = zerolog.Level(logLevelInt)
}
