package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// RootCmd is the root command for the CLI
	RootCmd = &cobra.Command{
		Use:   "rebost",
		Short: "Distributed Object Storage",
		Long:  "Distributed Object Storage easy to deploy",
	}
)

func init() {
	RootCmd.PersistentFlags().StringP("config", "c", "", "Configuration path")
	viper.BindPFlag("config", RootCmd.PersistentFlags().Lookup("config"))
}
