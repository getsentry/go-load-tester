/*
Copyright © 2021 Sentry

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

type runCliParams struct {
	port string
}

var runConfig runCliParams

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs the load tester",
	Long:  `Run the load tester either a worker or as a controller`,
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	},
	PreRun: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().StringVarP(&runConfig.port, "port", "p", "8000", "port to listen to")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func run(cmd *cobra.Command, args []string) {
	fmt.Println("Run in standalone mode")
	fmt.Printf("port is %s /n", runConfig.port)
}
