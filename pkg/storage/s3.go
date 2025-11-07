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
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rotisserie/eris"
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
	}

	S3Config struct {
		Bucket                 string
		Enabled                bool
		CreateMissingResources bool
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
	return &s3{
		awsCfg:   awscfg,
		client:   client,
		uploader: manager.NewUploader(client),
		cfg:      cfg,
		username: username,
	}, nil
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

	return nil
}

func (s *s3) Retrieve(ctx context.Context, request RetrieveFileRequest) (*fs.File, error) {
	if !s.cfg.Enabled {
		return nil, pkgerrors.NotImplementedError
	}
	if request.ToRetrieve == nil || request.Destination == nil {
		return nil, pkgerrors.NotImplementedError
	}
	// Construct key to match Store: {remoteDir}/{username}/{file.Dir}/{file.Name}
	// Store constructs: remoteDir (after addPrefix) + "/" + file.Dir + "/" + file.Name
	// where addPrefix adds username: remoteDir + "/" + username
	// For Retrieve, ToRetrieve.Dir should be structured as "{remoteDir}/{file.Dir}"
	// We split it: last component is file.Dir, everything before is remoteDir
	dirParts := strings.Split(request.ToRetrieve.Dir, "/")
	if len(dirParts) < 2 {
		// If there's only one component, treat it as file.Dir with empty remoteDir
		key := fmt.Sprintf("%s/%s/%s", s.username, request.ToRetrieve.Dir, request.ToRetrieve.Name)
		log.FromCtx(ctx).Info("Downloading file", zap.String("bucket", s.cfg.Bucket), zap.String("key", key))
		output, err := s.client.GetObject(ctx, &awss3.GetObjectInput{
			Bucket: aws.String(s.cfg.Bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return nil, err
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.FromCtx(ctx).Error("Failed to close body of output", zap.Error(err))
			}
		}(output.Body)

		// Open the destination file with create/overwrite permissions
		localFile, err := os.OpenFile(request.Destination.Absolute, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return nil, err
		}
		defer func() {
			err := localFile.Close()
			if err != nil {
				log.FromCtx(ctx).Error("Failed to close downloaded file", zap.Error(err))
			}
		}()

		// Write the downloaded content to the destination file
		_, err = io.Copy(localFile, output.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to write to temp file: %w", err)
		}

		f := fs.NewFile(request.Destination.Absolute, time.Now())
		// Override the Name to match the original file name, not the destination filename
		f.Name = request.ToRetrieve.Name
		return f, nil
	}

	// Split: last component is file.Dir, everything before is remoteDir
	fileDir := dirParts[len(dirParts)-1]
	remoteDir := strings.Join(dirParts[:len(dirParts)-1], "/")
	key := fmt.Sprintf("%s/%s/%s/%s", remoteDir, s.username, fileDir, request.ToRetrieve.Name)
	log.FromCtx(ctx).Info("Downloading file", zap.String("bucket", s.cfg.Bucket), zap.String("key", key))
	output, err := s.client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(s.cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.FromCtx(ctx).Error("Failed to close body of output", zap.Error(err))
		}
	}(output.Body)

	// Open the destination file with create/overwrite permissions
	localFile, err := os.OpenFile(request.Destination.Absolute, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := localFile.Close()
		if err != nil {
			log.FromCtx(ctx).Error("Failed to close downloaded file", zap.Error(err))
		}
	}()

	// Write the downloaded content to the destination file
	_, err = io.Copy(localFile, output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}

	f := fs.NewFile(request.Destination.Absolute, time.Now())
	// Override the Name to match the original file name, not the destination filename
	f.Name = request.ToRetrieve.Name

	return f, nil
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

	f, err := os.Open(file.Absolute)
	if err != nil {
		return eris.Wrap(err, "failed to open file")
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.FromCtx(ctx).Error("Failed to close file", zap.Error(err))
		}
	}()

	remoteDir, _ = strings.CutSuffix(remoteDir, "/")
	remoteDir = s.addPrefix(remoteDir)
	key := fmt.Sprintf("%s/%s", file.Dir, file.Name)
	if remoteDir != "" {
		key = fmt.Sprintf("%s/%s", remoteDir, key)
	}
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
		return eris.Wrap(err, "failed to upload")
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

func (s *s3) addPrefix(remoteDir string) string {
	return fmt.Sprintf("%s/%s", remoteDir, s.username)
}
