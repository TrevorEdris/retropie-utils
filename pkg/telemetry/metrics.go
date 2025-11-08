package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

var (
	// Sync metrics
	syncDuration       otelmetric.Float64Histogram
	syncFilesProcessed otelmetric.Int64Counter
	syncFilesTotal     otelmetric.Int64UpDownCounter
	syncOperationCount otelmetric.Int64Counter
	syncBytesTransferred otelmetric.Int64Counter

	// Storage metrics
	storageOperationDuration otelmetric.Float64Histogram
	storageOperationCount    otelmetric.Int64Counter
	storageBytesUploaded     otelmetric.Int64Counter
	storageBytesDownloaded   otelmetric.Int64Counter
	storageErrors            otelmetric.Int64Counter

	// File system metrics
	fsOperationDuration otelmetric.Float64Histogram
	fsFilesFound        otelmetric.Int64UpDownCounter

	// API metrics
	apiRequestDuration otelmetric.Float64Histogram
	apiRequestCount    otelmetric.Int64Counter
	apiRequestErrors   otelmetric.Int64Counter
)

// InitMetrics initializes all metric instruments
func InitMetrics() error {
	m := Meter()

	var err error

	// Sync metrics
	syncDuration, err = m.Float64Histogram(
		"syncer.sync.duration",
		otelmetric.WithDescription("Duration of sync operations in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	syncFilesProcessed, err = m.Int64Counter(
		"syncer.sync.files.processed",
		otelmetric.WithDescription("Total number of files processed during sync"),
	)
	if err != nil {
		return err
	}

	syncFilesTotal, err = m.Int64UpDownCounter(
		"syncer.sync.files.total",
		otelmetric.WithDescription("Total number of files found in source directory"),
	)
	if err != nil {
		return err
	}

	syncOperationCount, err = m.Int64Counter(
		"syncer.sync.operation.count",
		otelmetric.WithDescription("Total number of sync operations"),
	)
	if err != nil {
		return err
	}

	syncBytesTransferred, err = m.Int64Counter(
		"syncer.sync.bytes.transferred",
		otelmetric.WithDescription("Total bytes transferred during sync"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		return err
	}

	// Storage metrics
	storageOperationDuration, err = m.Float64Histogram(
		"syncer.storage.operation.duration",
		otelmetric.WithDescription("Duration of storage operations in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	storageOperationCount, err = m.Int64Counter(
		"syncer.storage.operation.count",
		otelmetric.WithDescription("Total number of storage operations"),
	)
	if err != nil {
		return err
	}

	storageBytesUploaded, err = m.Int64Counter(
		"syncer.storage.bytes.uploaded",
		otelmetric.WithDescription("Total bytes uploaded to storage"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		return err
	}

	storageBytesDownloaded, err = m.Int64Counter(
		"syncer.storage.bytes.downloaded",
		otelmetric.WithDescription("Total bytes downloaded from storage"),
		otelmetric.WithUnit("By"),
	)
	if err != nil {
		return err
	}

	storageErrors, err = m.Int64Counter(
		"syncer.storage.errors",
		otelmetric.WithDescription("Total number of storage errors"),
	)
	if err != nil {
		return err
	}

	// File system metrics
	fsOperationDuration, err = m.Float64Histogram(
		"syncer.fs.operation.duration",
		otelmetric.WithDescription("Duration of file system operations in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	fsFilesFound, err = m.Int64UpDownCounter(
		"syncer.fs.files.found",
		otelmetric.WithDescription("Number of files found in directory scan"),
	)
	if err != nil {
		return err
	}

	// API metrics
	apiRequestDuration, err = m.Float64Histogram(
		"syncer.api.request.duration",
		otelmetric.WithDescription("Duration of API requests in seconds"),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	apiRequestCount, err = m.Int64Counter(
		"syncer.api.request.count",
		otelmetric.WithDescription("Total number of API requests"),
	)
	if err != nil {
		return err
	}

	apiRequestErrors, err = m.Int64Counter(
		"syncer.api.request.errors",
		otelmetric.WithDescription("Total number of API request errors"),
	)
	if err != nil {
		return err
	}

	return nil
}

// RecordSyncDuration records the duration of a sync operation
func RecordSyncDuration(duration float64, fileType, status string) {
	if syncDuration != nil {
		syncDuration.Record(
			context.Background(),
			duration,
			otelmetric.WithAttributes(
				attribute.String("file_type", fileType),
				attribute.String("status", status),
			),
		)
	}
}

// RecordSyncFilesProcessed records files processed during sync
func RecordSyncFilesProcessed(count int64, fileType, action string) {
	if syncFilesProcessed != nil {
		syncFilesProcessed.Add(
			context.Background(),
			count,
			otelmetric.WithAttributes(
				attribute.String("file_type", fileType),
				attribute.String("action", action),
			),
		)
	}
}

// RecordSyncFilesTotal records the total number of files found
func RecordSyncFilesTotal(count int64, fileType string) {
	if syncFilesTotal != nil {
		syncFilesTotal.Add(
			context.Background(),
			count,
			otelmetric.WithAttributes(
				attribute.String("file_type", fileType),
			),
		)
	}
}

// RecordSyncOperation records a sync operation
func RecordSyncOperation(status string) {
	if syncOperationCount != nil {
		syncOperationCount.Add(
			context.Background(),
			1,
			otelmetric.WithAttributes(
				attribute.String("status", status),
			),
		)
	}
}

// RecordSyncBytesTransferred records bytes transferred during sync
func RecordSyncBytesTransferred(bytes int64, fileType, direction string) {
	if syncBytesTransferred != nil {
		syncBytesTransferred.Add(
			context.Background(),
			bytes,
			otelmetric.WithAttributes(
				attribute.String("file_type", fileType),
				attribute.String("direction", direction),
			),
		)
	}
}

// RecordStorageOperationDuration records the duration of a storage operation
func RecordStorageOperationDuration(duration float64, backend, operation, status string) {
	if storageOperationDuration != nil {
		storageOperationDuration.Record(
			context.Background(),
			duration,
			otelmetric.WithAttributes(
				attribute.String("backend", backend),
				attribute.String("operation", operation),
				attribute.String("status", status),
			),
		)
	}
}

// RecordStorageOperation records a storage operation
func RecordStorageOperation(backend, operation, status string) {
	if storageOperationCount != nil {
		storageOperationCount.Add(
			context.Background(),
			1,
			otelmetric.WithAttributes(
				attribute.String("backend", backend),
				attribute.String("operation", operation),
				attribute.String("status", status),
			),
		)
	}
}

// RecordStorageBytesUploaded records bytes uploaded to storage
func RecordStorageBytesUploaded(bytes int64, backend string) {
	if storageBytesUploaded != nil {
		storageBytesUploaded.Add(
			context.Background(),
			bytes,
			otelmetric.WithAttributes(
				attribute.String("backend", backend),
			),
		)
	}
}

// RecordStorageBytesDownloaded records bytes downloaded from storage
func RecordStorageBytesDownloaded(bytes int64, backend string) {
	if storageBytesDownloaded != nil {
		storageBytesDownloaded.Add(
			context.Background(),
			bytes,
			otelmetric.WithAttributes(
				attribute.String("backend", backend),
			),
		)
	}
}

// RecordStorageError records a storage error
func RecordStorageError(backend, errorType string) {
	if storageErrors != nil {
		storageErrors.Add(
			context.Background(),
			1,
			otelmetric.WithAttributes(
				attribute.String("backend", backend),
				attribute.String("error_type", errorType),
			),
		)
	}
}

// RecordFSOperationDuration records the duration of a file system operation
func RecordFSOperationDuration(duration float64, operation, status string) {
	if fsOperationDuration != nil {
		fsOperationDuration.Record(
			context.Background(),
			duration,
			otelmetric.WithAttributes(
				attribute.String("operation", operation),
				attribute.String("status", status),
			),
		)
	}
}

// RecordFSFilesFound records the number of files found
func RecordFSFilesFound(count int64, fileType string) {
	if fsFilesFound != nil {
		fsFilesFound.Add(
			context.Background(),
			count,
			otelmetric.WithAttributes(
				attribute.String("file_type", fileType),
			),
		)
	}
}

// RecordAPIRequestDuration records the duration of an API request
func RecordAPIRequestDuration(duration float64, endpoint, method, status string) {
	if apiRequestDuration != nil {
		apiRequestDuration.Record(
			context.Background(),
			duration,
			otelmetric.WithAttributes(
				attribute.String("endpoint", endpoint),
				attribute.String("method", method),
				attribute.String("status", status),
			),
		)
	}
}

// RecordAPIRequest records an API request
func RecordAPIRequest(endpoint, method, status string) {
	if apiRequestCount != nil {
		apiRequestCount.Add(
			context.Background(),
			1,
			otelmetric.WithAttributes(
				attribute.String("endpoint", endpoint),
				attribute.String("method", method),
				attribute.String("status", status),
			),
		)
	}
}

// RecordAPIRequestError records an API request error
func RecordAPIRequestError(endpoint, method, errorType string) {
	if apiRequestErrors != nil {
		apiRequestErrors.Add(
			context.Background(),
			1,
			otelmetric.WithAttributes(
				attribute.String("endpoint", endpoint),
				attribute.String("method", method),
				attribute.String("error_type", errorType),
			),
		)
	}
}

