package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	RootCmd = &cobra.Command{
		Use:   "rebost",
		Short: "Distributed Object Storage",
		Long:  "Distributed Object Storage easy to deploy",
	}
)

func init() {
	RootCmd.PersistentFlags().StringP("port", "p", "8000", "Destination port")
	viper.BindPFlag("port", RootCmd.PersistentFlags().Lookup("port"))

	RootCmd.PersistentFlags().StringP("config", "c", "", "Configuration path")
	viper.BindPFlag("config", RootCmd.PersistentFlags().Lookup("config"))
}
