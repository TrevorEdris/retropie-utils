package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/TrevorEdris/retropie-utils/pkg/config"
	"github.com/TrevorEdris/retropie-utils/pkg/fs"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type (
	s3 struct {
		awsCfg   awsconfig.Config
		uploader *manager.Uploader
		cfg      config.S3
	}
)

func NewS3Storage(cfg config.S3) (Storage, error) {
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
