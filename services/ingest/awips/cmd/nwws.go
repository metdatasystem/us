package main

import (
	"github.com/metdatasystem/us/services/ingest/awips/internal"
	"github.com/spf13/cobra"
)

var nwwsCmd = &cobra.Command{
	Use:   "nwws",
	Short: "Ingesting from the NWWS-OI",
	Long: `Ingest from the NOAA Weather Wire Service Open Interface.
	Listens to an XMPP room available from NOAA and makes the messages available to other MDS services.
	Requires NWWS credentials.`,
	Run: func(cmd *cobra.Command, args []string) {
		internal.NWWS(logLevel)
	},
}
