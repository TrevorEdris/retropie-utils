package storage

import (
	"context"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"
)

type (
	Storage interface {
		Init(ctx context.Context) error
		Store(ctx context.Context, remoteDir string, file *fs.File) error
		StoreAll(ctx context.Context, remoteDir string, files []*fs.File) error
	}
)
