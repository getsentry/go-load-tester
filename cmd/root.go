/*
Copyright © 2021 Sentry

*/
package cmd

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootConfig struct {
	cfgDirectory string
	useColor     bool
	logLevel     string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "go-load-tester",
	Short: "Load tester",
	Long: `Load tester utility based on vegeta. 
It supports multiple types of load tests for the Sentry infrastructure.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&rootConfig.cfgDirectory, "config", ".config", "configuration directory")
	rootCmd.PersistentFlags().StringVar(&rootConfig.logLevel, "log", "info", "Log level: trace, info, warn, (error), fatal, panic")
	rootCmd.PersistentFlags().BoolVar(&rootConfig.useColor, "color", false, "Use color (only for console output).")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// setup logging
	var consoleWriter = zerolog.ConsoleWriter{Out: os.Stdout, NoColor: !rootConfig.useColor,
		TimeFormat: "15:04:05"}
	log.Logger = zerolog.New(consoleWriter).With().Timestamp().Caller().Logger()

	var logLevel zerolog.Level

	switch strings.ToLower(rootConfig.logLevel) {
	case "t", "trc", "trace":
		logLevel = zerolog.TraceLevel
	case "d", "dbg", "debug":
		logLevel = zerolog.DebugLevel
	case "i", "inf", "info":
		logLevel = zerolog.InfoLevel
	case "w", "warn", "warning":
		logLevel = zerolog.WarnLevel
	case "e", "err", "error":
		logLevel = zerolog.ErrorLevel
	case "f", "fatal":
		logLevel = zerolog.FatalLevel
	case "p", "panic":
		logLevel = zerolog.PanicLevel
	case "dis", "disable", "disabled":
		logLevel = zerolog.Disabled
	default:
		logLevel = zerolog.ErrorLevel
	}

	zerolog.SetGlobalLevel(logLevel)

	viper.AddConfigPath(rootConfig.cfgDirectory)
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.SetEnvPrefix("LOAD_TEST")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Info().Msgf("Using config file:%s", viper.ConfigFileUsed())
	} else {
		log.Warn().Msg("Could not find config file")
	}

}
