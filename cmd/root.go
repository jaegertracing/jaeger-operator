package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jaegertracing/jaeger-operator/pkg/cmd/generate"
	"github.com/jaegertracing/jaeger-operator/pkg/cmd/start"
	"github.com/jaegertracing/jaeger-operator/pkg/cmd/version"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "jaeger-operator",
	Short: "The Kubernetes operator for Jaeger",
	Long:  `The Kubernetes operator for Jaeger`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jaeger-operator.yaml)")

	RootCmd.AddCommand(start.NewStartCommand())
	RootCmd.AddCommand(version.NewVersionCommand())
	RootCmd.AddCommand(generate.NewGenerateCommand())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".jaeger-operator" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".jaeger-operator")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
