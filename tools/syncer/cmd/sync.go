/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/TrevorEdris/retropie-utils/pkg/log"
	"github.com/TrevorEdris/retropie-utils/pkg/telemetry"
	"github.com/TrevorEdris/retropie-utils/tools/syncer/pkg/syncer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync files to a remote location",
	Long: `Sync files to a remote location.

The syncer will look at the configured RomsFolder
for any files matching a known file suffix, provided
the corresponding sync for that file type is enabled.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		ctx = log.ToCtx(ctx, log.FromCtx(ctx))

		// Initialize telemetry
		err := telemetry.Init(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize telemetry: %v\n", err)
			// Continue without telemetry
		} else {
			// Initialize metrics
			err = telemetry.InitMetrics()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to initialize metrics: %v\n", err)
			}
			// Setup graceful shutdown
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := telemetry.Shutdown(shutdownCtx); err != nil {
					fmt.Fprintf(os.Stderr, "Error shutting down telemetry: %v\n", err)
				}
			}()
		}

		cfg := syncer.Config{}
		err = viper.Unmarshal(&cfg)
		if err != nil {
			panic(err)
		}

		b, err := yaml.Marshal(cfg)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Running sync with config:\n%s", string(b))

		s, err := syncer.NewSyncer(ctx, cfg)
		if err != nil {
			panic(err)
		}
		err = s.Sync(ctx)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// syncCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// syncCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
