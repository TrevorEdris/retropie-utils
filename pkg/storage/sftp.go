package storage

import (
	"context"

	"github.com/TrevorEdris/retropie-utils/pkg/config"
	"github.com/TrevorEdris/retropie-utils/pkg/errors"
	"github.com/TrevorEdris/retropie-utils/pkg/fs"
)

type (
	sftp struct {
		cfg config.SFTP
	}
)

func NewSFTPStorage(cfg config.SFTP) (Storage, error) {
	return &sftp{cfg}, nil
}

func (s *sftp) Store(ctx context.Context, file *fs.File) error {
	return errors.NotImplementedError
}

func (s *sftp) StoreAll(ctx context.Context, file []*fs.File) error {
	return errors.NotImplementedError
}
