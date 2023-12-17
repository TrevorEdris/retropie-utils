package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type (
	s3 struct {
		awsCfg   config.Config
		uploader *manager.Uploader
		cfg      S3Config
	}

	S3Config struct {
		Enabled bool
		Bucket  string
	}
)

var _ Storage = &s3{}

func NewS3Storage(cfg S3Config) (Storage, error) {
	awscfg, err := newAwsConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	return &s3{
		awsCfg:   awscfg,
		uploader: manager.NewUploader(awss3.NewFromConfig(awscfg)),
		cfg:      cfg,
	}, nil
}

func (s *s3) Store(ctx context.Context, file *fs.File) error {
	if !s.cfg.Enabled {
		return nil
	}

	f, err := os.Open(file.Absolute)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = s.uploader.Upload(
		ctx,
		&awss3.PutObjectInput{
			Bucket: aws.String(s.cfg.Bucket),
			Key:    aws.String(fmt.Sprintf("%s/%s", file.Dir, file.Name)),
			Body:   f,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *s3) StoreAll(ctx context.Context, files []*fs.File) error {
	for _, f := range files {
		err := s.Store(ctx, f)
		if err != nil {
			return err
		}
	}
	return nil
}
