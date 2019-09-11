package cmd

import (
	"github.com/masudur-rahman/apiserver-prom-exporter/api"
	"github.com/spf13/cobra"
)

var port string
var bypass bool
var stopTime int16

var startApp = &cobra.Command{
	Use:   "start",
	Short: "Start the app",
	Long:  "This starts the apiserver-prom-exporter",
	Run: func(cmd *cobra.Command, args []string) {
		api.AssignValues(port, bypass, stopTime)
		api.StartTheApp()
	},
}

func init() {
	startApp.PersistentFlags().StringVarP(&port, "port", "p", "9999", "port number for the server")
	startApp.PersistentFlags().BoolVarP(&bypass, "bypass", "b", false, "Bypass authentication parameter")
	startApp.PersistentFlags().Int16VarP(&stopTime, "stopTime", "s", 0, "The time after which the server will stop")

	rootCmd.AddCommand(startApp)
}
