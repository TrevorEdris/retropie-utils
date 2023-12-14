package storage

import (
	"context"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"
)

type (
	Storage interface {
		Store(ctx context.Context, file *fs.File) error
		StoreAll(ctx context.Context, files []*fs.File) error
	}
)
