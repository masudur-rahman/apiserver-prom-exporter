package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Version of apiserver-prom-exporter",
	Long:  "The version of the apiserver-prom-exporter app is",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("apiserver-prom-exporter - v2.0")
	},
}
