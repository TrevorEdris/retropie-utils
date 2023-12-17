package storage

import (
	"context"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"
)

type (
	Storage interface {
		Store(ctx context.Context, remoteDir string, file *fs.File) error
		StoreAll(ctx context.Context, remoteDir string, files []*fs.File) error
	}
)
