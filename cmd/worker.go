/*
Copyright © 2021 Sentry

*/
package cmd

import (
	"fmt"
	"github.com/getsentry/go-load-tester/web_server"
	"github.com/spf13/cobra"
)

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Run a worker, that waits for commands from a server",
	Long:  `Runs in worker mode waiting to execute commands sent via the command endpoint`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("worker called")
		fmt.Printf("port is %s \n", runConfig.port)
		web_server.RunWebServer(runConfig.port, runConfig.targetUrl)
	},
}

func init() {
	runCmd.AddCommand(workerCmd)
}
