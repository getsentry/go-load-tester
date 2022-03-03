/*
Copyright © 2021 Sentry

*/
package cmd

import (
	"github.com/spf13/cobra"
)

type runCliParams struct {
	port      string
	targetUrl string
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
	runCmd.PersistentFlags().StringVarP(&runConfig.targetUrl, "target-url", "t", "", "target URL for the attack")
}
