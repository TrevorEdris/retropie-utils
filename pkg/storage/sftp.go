package storage

import (
	"context"

	"github.com/TrevorEdris/retropie-utils/pkg/errors"
	"github.com/TrevorEdris/retropie-utils/pkg/fs"
)

type (
	sftp struct {
		cfg SFTPConfig
	}

	SFTPConfig struct {
		Enabled   bool
		Username  string
		Password  string
		Port      int
		RemoteDir string
	}
)

var _ Storage = &sftp{}

func NewSFTPStorage(cfg SFTPConfig) (Storage, error) {
	return &sftp{cfg}, nil
}

func (s *sftp) Init(ctx context.Context) error {
	return errors.NotImplementedError
}

func (s *sftp) Store(ctx context.Context, remoteDir string, file *fs.File) error {
	return errors.NotImplementedError
}

func (s *sftp) StoreAll(ctx context.Context, remoteDir string, file []*fs.File) error {
	return errors.NotImplementedError
}
