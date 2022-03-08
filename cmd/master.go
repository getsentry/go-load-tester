/*
Copyright © 2021 Sentry

*/
package cmd

import (
	"github.com/getsentry/go-load-tester/web_server"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// master runs the load tester in master mode.
var master = &cobra.Command{
	Use:   "master",
	Short: "Run load tester in master mode.",
	Long: `Runs the load tester in master mode. 
In master mode the load tester accepts registrations from workers.
Every command it receives it distributes to the workers.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msgf("Running load tester in master mode at port: %s", runConfig.port)
		web_server.RunMasterWebServer(runConfig.port, runConfig.statsdAddr)
	},
}

func init() {
	runCmd.AddCommand(master)
}
