package main

import (
	"github.com/metdatasystem/us/services/parse/awips/internal"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Parsing from Kafka",
	Run: func(cmd *cobra.Command, args []string) {
		internal.Server(logLevel)
	},
}
