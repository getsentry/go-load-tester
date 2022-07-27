/*
Copyright © 2021 Sentry

*/
package cmd

import (
	"github.com/getsentry/go-load-tester/utils"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/getsentry/go-load-tester/web_server"
)

var runWorkerParams struct {
	masterUrl        string
	projectsFileName string
}

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Run a worker, that waits for commands from a server",
	Long:  `Runs in worker mode waiting to execute commands sent via the command endpoint`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msgf("Running load tester in worker mode at port: %s", runConfig.port)
		utils.RegisterProjectProvider(runWorkerParams.projectsFileName)
		web_server.RunWorkerWebServer(runConfig.port, runConfig.targetUrl, runWorkerParams.masterUrl, runConfig.statsdAddr)
	},
}

func init() {
	runCmd.AddCommand(workerCmd)
	workerCmd.Flags().StringVarP(&runWorkerParams.masterUrl, "master-url", "m", "", "Registers worker with the specified master")
	workerCmd.Flags().StringVar(&runWorkerParams.projectsFileName, "projects-file", "", "Path to the projects file")
}
