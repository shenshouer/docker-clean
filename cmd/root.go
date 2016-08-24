package cmd

import (
	"fmt"
	"os"

	"docker-clean/clean"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "docker-clean",
	Short: "A brief description of your application",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if err := clean.Clean(cmd, args); err != nil {
			log.Fatal(err)
		}
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	// RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.docker-clean.yaml)")
	RootCmd.PersistentFlags().StringP("docker-host", "", "", "for control the docker daemon. if set volume to /var/run/docker.sock, this option will not take effect")
	RootCmd.PersistentFlags().StringSliceP("exclude-images", "", []string{}, "images to exclude, --exclude-images image:tag [--exclude-images image1:tag1]")
	RootCmd.PersistentFlags().StringP("start-time", "", "2:00", "start time for start task")
	RootCmd.PersistentFlags().StringP("stop-time", "", "6:00", "stop time for stop task")
	// RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".docker-clean") // name of config file (without extension)
	viper.AddConfigPath("$HOME")         // adding home directory as first search path
	viper.AutomaticEnv()                 // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
