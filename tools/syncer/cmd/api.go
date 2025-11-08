package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TrevorEdris/retropie-utils/pkg/log"
	"github.com/TrevorEdris/retropie-utils/pkg/telemetry"
	"github.com/TrevorEdris/retropie-utils/tools/syncer/pkg/api"
	"github.com/TrevorEdris/retropie-utils/tools/syncer/pkg/syncer"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var apiPort int

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Start the API server",
	Long: `Start the API server that provides HTTP endpoints to trigger sync operations.

The API server will run continuously and listen for HTTP requests.
Use the /sync endpoint to trigger a sync operation.`,
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

		// Load configuration
		cfg := syncer.Config{}
		err = viper.Unmarshal(&cfg)
		if err != nil {
			log.FromCtx(ctx).Fatal("Failed to unmarshal config", zap.Error(err))
		}

		// Create syncer instance
		s, err := syncer.NewSyncer(ctx, cfg)
		if err != nil {
			log.FromCtx(ctx).Fatal("Failed to create syncer", zap.Error(err))
		}

		// Create and start API server
		server := api.NewServer(apiPort, s)

		// Start metrics server on port 9090 if Prometheus exporter is available
		var metricsServer *http.Server
		reg := telemetry.GetPrometheusRegistry()
		if reg != nil {
			metricsMux := http.NewServeMux()
			metricsMux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
			metricsServer = &http.Server{
				Addr:         ":9090",
				Handler:      metricsMux,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
			}
			go func() {
				if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.FromCtx(ctx).Error("Metrics server error", zap.Error(err))
				}
			}()
			log.FromCtx(ctx).Info("Metrics server started", zap.Int("port", 9090))
		}

		// Setup graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Start API server in a goroutine
		serverErrChan := make(chan error, 1)
		go func() {
			if err := server.Start(ctx); err != nil {
				serverErrChan <- err
			}
		}()

		log.FromCtx(ctx).Info("API server started", zap.Int("port", apiPort))

		// Wait for interrupt or server error
		select {
		case <-sigChan:
			log.FromCtx(ctx).Info("Received shutdown signal, shutting down gracefully...")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				log.FromCtx(ctx).Error("Error shutting down API server", zap.Error(err))
			}
			if metricsServer != nil {
				if err := metricsServer.Shutdown(shutdownCtx); err != nil {
					log.FromCtx(ctx).Error("Error shutting down metrics server", zap.Error(err))
				}
			}
		case err := <-serverErrChan:
			log.FromCtx(ctx).Fatal("Server error", zap.Error(err))
		}
	},
}

func init() {
	rootCmd.AddCommand(apiCmd)
	apiCmd.Flags().IntVarP(&apiPort, "port", "p", 8000, "Port to listen on")
}
