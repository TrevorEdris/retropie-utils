package storage

import (
	"context"

	"github.com/TrevorEdris/retropie-utils/pkg/config"
	"github.com/TrevorEdris/retropie-utils/pkg/errors"
	"github.com/TrevorEdris/retropie-utils/pkg/fs"
)

type (
	gdrive struct {
		cfg config.GoogleDrive
	}
)

func NewGoogleDriveStorage(cfg config.GoogleDrive) (Storage, error) {
	return &gdrive{cfg}, nil
}

func (g *gdrive) Store(ctx context.Context, file *fs.File) error {
	return errors.NotImplementedError
}

func (g *gdrive) StoreAll(ctx context.Context, file []*fs.File) error {
	return errors.NotImplementedError
}
