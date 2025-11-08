package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	pkgerrors "github.com/TrevorEdris/retropie-utils/pkg/errors"
	"github.com/TrevorEdris/retropie-utils/pkg/fs"
	"github.com/TrevorEdris/retropie-utils/pkg/log"
	"github.com/TrevorEdris/retropie-utils/pkg/telemetry"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rotisserie/eris"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type (
	s3 struct {
		awsCfg             config.Config
		client             *awss3.Client
		uploader           *manager.Uploader
		cfg                S3Config
		resourcesValidated bool
		username           string
		dynamodbClient     *DynamoDBClient
	}

	S3Config struct {
		Bucket                 string
		Enabled                bool
		CreateMissingResources bool
		DynamoDB               DynamoDBConfig
	}
)

var _ Storage = &s3{}

func NewS3Storage(ctx context.Context, cfg S3Config, username string) (Storage, error) {
	awscfg, err := newAwsConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := awss3.NewFromConfig(awscfg, func(o *awss3.Options) {
		o.UsePathStyle = true
	})

	s3Storage := &s3{
		awsCfg:   awscfg,
		client:   client,
		uploader: manager.NewUploader(client),
		cfg:      cfg,
		username: username,
	}

	// Initialize DynamoDB client if enabled
	if cfg.DynamoDB.Enabled {
		dynamodbClient, err := NewDynamoDBClient(ctx, cfg.DynamoDB, username)
		if err != nil {
			return nil, eris.Wrap(err, "failed to create DynamoDB client")
		}
		s3Storage.dynamodbClient = dynamodbClient
	}

	return s3Storage, nil
}

func (s *s3) Init(ctx context.Context) error {
	// Validate required S3 resources exist
	exist, err := s.checkIfResourcesExist(ctx)
	if err != nil {
		return err
	}

	// If they do not exist, create them if config is enabled
	if !exist && s.cfg.CreateMissingResources {
		err = s.createMissingResources(ctx)
		if err != nil {
			return err
		}
		s.resourcesValidated = true
	}

	// Initialize DynamoDB if enabled
	if s.dynamodbClient != nil {
		err = s.dynamodbClient.Init(ctx)
		if err != nil {
			return eris.Wrap(err, "failed to initialize DynamoDB")
		}
	}

	return nil
}

func (s *s3) Retrieve(ctx context.Context, request RetrieveFileRequest) (*fs.File, error) {
	if !s.cfg.Enabled {
		return nil, pkgerrors.NotImplementedError
	}
	if request.ToRetrieve == nil || request.Destination == nil {
		return nil, pkgerrors.NotImplementedError
	}

	startTime := time.Now()
	ctx, span := telemetry.Tracer().Start(ctx, "syncer.storage.retrieve")
	defer span.End()
	span.SetAttributes(
		attribute.String("syncer.storage.backend", "s3"),
		attribute.String("syncer.storage.operation", "retrieve"),
		attribute.String("syncer.storage.bucket", s.cfg.Bucket),
	)

	var key string

	// If DynamoDB is enabled, look up the S3 location from metadata
	if s.dynamodbClient != nil {
		metadata, err := s.dynamodbClient.GetFileMetadataByFile(ctx, request.ToRetrieve)
		if err != nil {
			span.RecordError(err)
			telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "s3", "retrieve", "error")
			telemetry.RecordStorageOperation("s3", "retrieve", "error")
			return nil, eris.Wrap(err, "failed to get file metadata from DynamoDB")
		}
		if metadata != nil && metadata.S3Location != "" {
			// Extract just the key from the full S3 location (remove bucket if present)
			key = metadata.S3Location
			log.FromCtx(ctx).Info("Found S3 location in DynamoDB", zap.String("key", key))
		} else {
			// Fall back to constructing key if not found in DynamoDB
			log.FromCtx(ctx).Warn("File not found in DynamoDB, falling back to key construction", zap.String("file", request.ToRetrieve.Name))
			key = s.constructKey(request.ToRetrieve)
		}
	} else {
		// Fall back to original behavior: construct key from request
		key = s.constructKey(request.ToRetrieve)
	}

	span.SetAttributes(attribute.String("syncer.storage.key", key))
	log.FromCtx(ctx).Info("Downloading file", zap.String("bucket", s.cfg.Bucket), zap.String("key", key))
	output, err := s.client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		span.RecordError(err)
		telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "s3", "retrieve", "error")
		telemetry.RecordStorageOperation("s3", "retrieve", "error")
		telemetry.RecordStorageError("s3", "download_failed")
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.FromCtx(ctx).Error("Failed to close body of output", zap.Error(err))
		}
	}(output.Body)

	// Get file size from ContentLength if available
	fileSize := int64(0)
	if output.ContentLength != nil {
		fileSize = *output.ContentLength
		span.SetAttributes(attribute.Int64("syncer.storage.file.size", fileSize))
	}

	// Open the destination file with create/overwrite permissions
	localFile, err := os.OpenFile(request.Destination.Absolute, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		span.RecordError(err)
		telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "s3", "retrieve", "error")
		telemetry.RecordStorageOperation("s3", "retrieve", "error")
		return nil, err
	}
	defer func() {
		err := localFile.Close()
		if err != nil {
			log.FromCtx(ctx).Error("Failed to close downloaded file", zap.Error(err))
		}
	}()

	// Write the downloaded content to the destination file
	bytesWritten, err := io.Copy(localFile, output.Body)
	if err != nil {
		span.RecordError(err)
		telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "s3", "retrieve", "error")
		telemetry.RecordStorageOperation("s3", "retrieve", "error")
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}

	telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "s3", "retrieve", "success")
	telemetry.RecordStorageOperation("s3", "retrieve", "success")
	telemetry.RecordStorageBytesDownloaded(bytesWritten, "s3")

	f := fs.NewFile(request.Destination.Absolute, time.Now())
	// Override the Name to match the original file name, not the destination filename
	f.Name = request.ToRetrieve.Name

	return f, nil
}

// constructKey constructs the S3 key from the retrieve request (fallback behavior)
func (s *s3) constructKey(toRetrieve *fs.File) string {
	// Construct key to match Store: {remoteDir}/{username}/{file.Dir}/{file.Name}
	// Store constructs: remoteDir (after addPrefix) + "/" + file.Dir + "/" + file.Name
	// where addPrefix adds username: remoteDir + "/" + username
	// For Retrieve, ToRetrieve.Dir should be structured as "{remoteDir}/{file.Dir}"
	// We split it: last component is file.Dir, everything before is remoteDir
	dirParts := strings.Split(toRetrieve.Dir, "/")
	if len(dirParts) < 2 {
		// If there's only one component, treat it as file.Dir with empty remoteDir
		return fmt.Sprintf("%s/%s/%s", s.username, toRetrieve.Dir, toRetrieve.Name)
	}

	// Split: last component is file.Dir, everything before is remoteDir
	fileDir := dirParts[len(dirParts)-1]
	remoteDir := strings.Join(dirParts[:len(dirParts)-1], "/")
	return fmt.Sprintf("%s/%s/%s/%s", remoteDir, s.username, fileDir, toRetrieve.Name)
}

func (s *s3) checkIfResourcesExist(ctx context.Context) (bool, error) {
	_, err := s.client.HeadBucket(ctx, &awss3.HeadBucketInput{
		Bucket: aws.String(s.cfg.Bucket),
	})
	if err == nil {
		log.FromCtx(ctx).Info("Bucket exists", zap.String("bucket", s.cfg.Bucket))
		return true, nil
	}

	var notFoundErr *types.NotFound
	if errors.As(err, &notFoundErr) {
		return false, nil
	}
	return false, err
}

func (s *s3) createMissingResources(ctx context.Context) error {
	_, err := s.client.CreateBucket(
		ctx,
		&awss3.CreateBucketInput{
			Bucket: aws.String(s.cfg.Bucket),
		})
	if err != nil {
		log.FromCtx(ctx).Error("Failed to create bucket", zap.String("bucket", s.cfg.Bucket), zap.Error(err))
		return err
	}
	log.FromCtx(ctx).Info("Successfully created bucket", zap.String("bucket", s.cfg.Bucket))
	return nil
}

func (s *s3) Store(ctx context.Context, remoteDir string, file *fs.File) error {
	if !s.cfg.Enabled {
		return nil
	}

	startTime := time.Now()
	ctx, span := telemetry.Tracer().Start(ctx, "syncer.storage.store")
	defer span.End()
	span.SetAttributes(
		attribute.String("syncer.storage.backend", "s3"),
		attribute.String("syncer.storage.operation", "store"),
		attribute.String("syncer.storage.bucket", s.cfg.Bucket),
	)

	f, err := os.Open(file.Absolute)
	if err != nil {
		span.RecordError(err)
		telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "s3", "store", "error")
		telemetry.RecordStorageOperation("s3", "store", "error")
		return eris.Wrap(err, "failed to open file")
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.FromCtx(ctx).Error("Failed to close file", zap.Error(err))
		}
	}()

	// Get file size for metrics
	fileInfo, _ := f.Stat()
	fileSize := int64(0)
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	remoteDir, _ = strings.CutSuffix(remoteDir, "/")
	remoteDir = s.addPrefix(remoteDir)
	key := fmt.Sprintf("%s/%s", file.Dir, file.Name)
	if remoteDir != "" {
		key = fmt.Sprintf("%s/%s", remoteDir, key)
	}
	span.SetAttributes(
		attribute.String("syncer.storage.key", key),
		attribute.Int64("syncer.storage.file.size", fileSize),
	)
	log.FromCtx(ctx).Sugar().Infof("Uploading %s to %s/%s", file.Absolute, s.cfg.Bucket, key)

	_, err = s.uploader.Upload(
		ctx,
		&awss3.PutObjectInput{
			Bucket: aws.String(s.cfg.Bucket),
			Key:    aws.String(key),
			Body:   f,
		},
	)
	if err != nil {
		span.RecordError(err)
		telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "s3", "store", "error")
		telemetry.RecordStorageOperation("s3", "store", "error")
		telemetry.RecordStorageError("s3", "upload_failed")
		return eris.Wrap(err, "failed to upload")
	}

	telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "s3", "store", "success")
	telemetry.RecordStorageOperation("s3", "store", "success")
	telemetry.RecordStorageBytesUploaded(fileSize, "s3")

	// Store metadata in DynamoDB if enabled
	if s.dynamodbClient != nil {
		err = s.dynamodbClient.StoreFileMetadata(ctx, key, file, remoteDir)
		if err != nil {
			// Log error but don't fail the upload
			log.FromCtx(ctx).Error("Failed to store file metadata in DynamoDB", zap.Error(err), zap.String("key", key))
			telemetry.RecordStorageError("dynamodb", "metadata_store_failed")
			// Optionally return error if you want to fail the upload on DynamoDB failure
			// return eris.Wrap(err, "failed to store file metadata")
		}
	}

	return nil
}

func (s *s3) StoreAll(ctx context.Context, remoteDir string, files []*fs.File) error {
	for _, f := range files {
		err := s.Store(ctx, remoteDir, f)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *s3) GetFileLastModified(ctx context.Context, remoteDir string, file *fs.File) (*time.Time, error) {
	if !s.cfg.Enabled {
		return nil, pkgerrors.NotImplementedError
	}

	startTime := time.Now()
	ctx, span := telemetry.Tracer().Start(ctx, "syncer.storage.get_metadata")
	defer span.End()
	span.SetAttributes(
		attribute.String("syncer.storage.backend", "s3"),
		attribute.String("syncer.storage.operation", "get_metadata"),
	)

	// Construct the S3 key the same way Store does
	remoteDir, _ = strings.CutSuffix(remoteDir, "/")
	remoteDir = s.addPrefix(remoteDir)
	key := fmt.Sprintf("%s/%s", file.Dir, file.Name)
	if remoteDir != "" {
		key = fmt.Sprintf("%s/%s", remoteDir, key)
	}
	span.SetAttributes(attribute.String("syncer.storage.key", key))

	// First, try to get metadata from DynamoDB if enabled
	if s.dynamodbClient != nil {
		metadata, err := s.dynamodbClient.GetFileMetadataByFile(ctx, file)
		if err == nil && metadata != nil && metadata.LastModifiedTime > 0 {
			lastModified := time.UnixMilli(metadata.LastModifiedTime)
			telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "dynamodb", "get_metadata", "success")
			telemetry.RecordStorageOperation("dynamodb", "get_metadata", "success")
			return &lastModified, nil
		}
		// If DynamoDB lookup fails or returns nil, fall through to S3 HeadObject
	}

	// Use HeadObject to get metadata from S3
	output, err := s.client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFoundErr *types.NotFound
		if errors.As(err, &notFoundErr) {
			// File doesn't exist in S3
			telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "s3", "get_metadata", "not_found")
			telemetry.RecordStorageOperation("s3", "get_metadata", "not_found")
			return nil, nil
		}
		span.RecordError(err)
		telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "s3", "get_metadata", "error")
		telemetry.RecordStorageOperation("s3", "get_metadata", "error")
		telemetry.RecordStorageError("s3", "head_object_failed")
		return nil, eris.Wrap(err, "failed to get file metadata from S3")
	}

	if output.LastModified == nil {
		telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "s3", "get_metadata", "success")
		telemetry.RecordStorageOperation("s3", "get_metadata", "success")
		return nil, nil
	}

	telemetry.RecordStorageOperationDuration(time.Since(startTime).Seconds(), "s3", "get_metadata", "success")
	telemetry.RecordStorageOperation("s3", "get_metadata", "success")
	return output.LastModified, nil
}

func (s *s3) addPrefix(remoteDir string) string {
	return fmt.Sprintf("%s/%s", remoteDir, s.username)
}
