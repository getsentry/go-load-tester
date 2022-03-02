/*
Copyright © 2021 Sentry

*/
package cmd

import (
	"fmt"
	"github.com/getsentry/go-load-tester/web_server"

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
		fmt.Println("master called")
		web_server.RunMasterWebServer(runConfig.port)
	},
}

func init() {
	runCmd.AddCommand(master)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// master.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// master.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
