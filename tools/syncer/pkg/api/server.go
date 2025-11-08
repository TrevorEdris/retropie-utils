package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/TrevorEdris/retropie-utils/pkg/log"
	"github.com/TrevorEdris/retropie-utils/pkg/telemetry"
	"github.com/TrevorEdris/retropie-utils/tools/syncer/pkg/syncer"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type Server struct {
	syncerInstance syncer.Syncer
	mu             sync.Mutex
	isRunning      bool
	lastSyncTime   time.Time
	lastSyncError  error
	port           int
	server         *http.Server
}

type SyncResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	StartTime time.Time `json:"start_time,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type StatusResponse struct {
	IsRunning    bool      `json:"is_running"`
	LastSyncTime time.Time `json:"last_sync_time,omitempty"`
	LastError    string    `json:"last_error,omitempty"`
}

func NewServer(port int, syncerInstance syncer.Syncer) *Server {
	return &Server{
		syncerInstance: syncerInstance,
		port:           port,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.withTelemetry("/health", http.MethodGet, s.handleHealth))
	mux.HandleFunc("/sync", s.withTelemetry("/sync", http.MethodPost, s.handleSync))
	mux.HandleFunc("/status", s.withTelemetry("/status", http.MethodGet, s.handleStatus))

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.FromCtx(ctx).Info("Starting API server", zap.Int("port", s.port))
	
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

// withTelemetry wraps an HTTP handler with telemetry instrumentation
func (s *Server) withTelemetry(endpoint, expectedMethod string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		ctx := r.Context()
		ctx = log.ToCtx(ctx, log.FromCtx(ctx))

		// Start trace span
		ctx, span := telemetry.Tracer().Start(ctx, "syncer.api.request")
		defer span.End()
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", endpoint),
			attribute.String("http.url", r.URL.Path),
		)

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Check method
		if r.Method != expectedMethod {
			rw.statusCode = http.StatusMethodNotAllowed
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			statusStr := strconv.Itoa(rw.statusCode)
			telemetry.RecordAPIRequest(endpoint, r.Method, statusStr)
			telemetry.RecordAPIRequestDuration(time.Since(startTime).Seconds(), endpoint, r.Method, statusStr)
			telemetry.RecordAPIRequestError(endpoint, r.Method, "method_not_allowed")
			span.SetAttributes(
				attribute.Int("http.status_code", rw.statusCode),
				attribute.String("error", "method_not_allowed"),
			)
			span.RecordError(fmt.Errorf("method not allowed: %s", r.Method))
			return
		}

		// Call the actual handler
		handler(rw, r)

		// Record metrics
		duration := time.Since(startTime).Seconds()
		statusStr := strconv.Itoa(rw.statusCode)
		telemetry.RecordAPIRequest(endpoint, r.Method, statusStr)
		telemetry.RecordAPIRequestDuration(duration, endpoint, r.Method, statusStr)

		// Record error if status code indicates error
		if rw.statusCode >= 400 {
			telemetry.RecordAPIRequestError(endpoint, r.Method, fmt.Sprintf("http_%d", rw.statusCode))
			span.RecordError(fmt.Errorf("HTTP error: %d", rw.statusCode))
		}

		span.SetAttributes(
			attribute.Int("http.status_code", rw.statusCode),
			attribute.Float64("http.duration", duration),
		)
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = log.ToCtx(ctx, log.FromCtx(ctx))

	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(SyncResponse{
			Success: false,
			Message: "Sync operation already in progress",
		})
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	startTime := time.Now()

	// Run sync in a goroutine so we can return immediately
	// Create a new context independent of the HTTP request context
	// to avoid cancellation when the HTTP request completes
	syncCtx := context.Background()
	syncCtx = log.ToCtx(syncCtx, log.FromCtx(ctx)) // Preserve logger from request context
	
	go func() {
		defer func() {
			s.mu.Lock()
			s.isRunning = false
			s.lastSyncTime = startTime
			s.mu.Unlock()
		}()

		err := s.syncerInstance.Sync(syncCtx)
		if err != nil {
			s.mu.Lock()
			s.lastSyncError = err
			s.mu.Unlock()
			log.FromCtx(syncCtx).Error("Sync operation failed", zap.Error(err))
		} else {
			s.mu.Lock()
			s.lastSyncError = nil
			s.mu.Unlock()
			log.FromCtx(syncCtx).Info("Sync operation completed successfully")
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(SyncResponse{
		Success:   true,
		Message:   "Sync operation started",
		StartTime: startTime,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	status := StatusResponse{
		IsRunning:    s.isRunning,
		LastSyncTime: s.lastSyncTime,
	}
	if s.lastSyncError != nil {
		status.LastError = s.lastSyncError.Error()
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

