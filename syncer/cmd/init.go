/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrevorEdris/retropie-utils/pkg/config"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize an example configuration for use with syncer",
	Long: `Initialize an example configuration for use with syncer.

A file $HOME/.syncer/config.example.yaml will be created, showing
a working* example of a configuration.

**Note:** Run the following command to copy the example file to
the actual configuration file.

**Note:** The example configuration file will result in the syncer
[loading the default config for AWS](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#loading-aws-shared-configuration).

cp $HOME/.syncer/config.example.yaml $HOME/.syncer/config.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Unable to determine user home directory: %s", err)
			os.Exit(1)
		}
		syncerDir := filepath.Join(home, ".syncer")
		err = config.CreateExample(syncerDir)
		if err != nil {
			fmt.Printf("Unable to create example configuration: %s", err)
		}
	},
}

func init() {
	configCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
