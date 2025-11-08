package cmd

import (
	"context"
	"net/http"
	"time"

	"github.com/TrevorEdris/retropie-utils/pkg/telemetry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Start metrics server",
	Long:  `Start an HTTP server to expose Prometheus metrics`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		
		// Initialize telemetry
		err := telemetry.Init(ctx)
		if err != nil {
			panic(err)
		}
		
		// Initialize metrics
		err = telemetry.InitMetrics()
		if err != nil {
			panic(err)
		}
		
		// Get Prometheus registry
		reg := telemetry.GetPrometheusRegistry()
		if reg == nil {
			panic("Prometheus registry not initialized")
		}
		
		// Create Prometheus HTTP handler
		handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
		
		// Start metrics server
		http.Handle("/metrics", handler)
		server := &http.Server{
			Addr:         ":9090",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				panic(err)
			}
		}()
		
		// Keep running
		select {}
	},
}

func init() {
	rootCmd.AddCommand(metricsCmd)
}

