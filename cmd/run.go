/*
Copyright © 2021 Sentry
*/
package cmd

import (
	"github.com/spf13/cobra"
)

type runCliParams struct {
	port       string
	targetUrl  string
	statsdAddr string
	workers    int
}

var runConfig runCliParams

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs the load tester",
	Long:  `Run the load tester either a worker or as a controller`,
	PreRun: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().StringVarP(&runConfig.port, "port", "p", "8000", "port to listen to")
	runCmd.PersistentFlags().IntVarP(&runConfig.workers, "workers", "w", 1, "threads to use to build load")
	runCmd.PersistentFlags().StringVarP(&runConfig.targetUrl, "target-url", "t", "", "target URL for the attack")
	runCmd.PersistentFlags().StringVar(&runConfig.statsdAddr, "statsd-server", "", "ip:port for the statsd server")
}
