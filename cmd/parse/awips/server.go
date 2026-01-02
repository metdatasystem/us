package main

import (
	internal "github.com/metdatasystem/us/internal/parse/awips"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Parsing from Kafka",
	Run: func(cmd *cobra.Command, args []string) {
		internal.Server(logLevel)
	},
}
