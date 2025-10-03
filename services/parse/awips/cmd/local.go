package main

import (
	"github.com/metdatasystem/us/services/parse/awips/internal"
	"github.com/spf13/cobra"
)

var path string

var localCmd = &cobra.Command{
	Use:   "local",
	Short: "Parsing from local files and directories",
	Run: func(cmd *cobra.Command, args []string) {
		internal.Local(path, logLevel)
	},
}

func init() {
	localCmd.Flags().StringVarP(&path, "path", "p", ".", "The path to read from. Can be a file or a directory.")
}
