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
	Use:   "controller",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("controller called")
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
