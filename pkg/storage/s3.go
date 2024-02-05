package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"
	"github.com/TrevorEdris/retropie-utils/pkg/log"
	"github.com/rotisserie/eris"
	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type (
	s3 struct {
		awsCfg             config.Config
		client             *awss3.Client
		uploader           *manager.Uploader
		cfg                S3Config
		resourcesValidated bool
	}

	S3Config struct {
		Bucket                 string
		Enabled                bool
		CreateMissingResources bool
	}
)

var _ Storage = &s3{}

func NewS3Storage(ctx context.Context, cfg S3Config) (Storage, error) {
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
	defer f.Close()

	remoteDir, _ = strings.CutSuffix(remoteDir, "/")
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
