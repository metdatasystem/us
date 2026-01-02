package main

import (
	internal "github.com/metdatasystem/us/internal/parse/awips"
	"github.com/spf13/cobra"
)

var path string

var localCmd = &cobra.Command{
	Use:   "local",
	Short: "Ingesting from local files and directories",
	Long: `Ingest from the NOAA Weather Wire Service Open Interface.
	Listens to an XMPP room available from NOAA and makes the messages available to other MDS services.
	Requires NWWS credentials.`,
	Run: func(cmd *cobra.Command, args []string) {
		internal.Local(path, logLevel)
	},
}

func init() {
	localCmd.Flags().StringVarP(&path, "path", "p", ".", "The path to read from. Can be a file or a directory.")
}
