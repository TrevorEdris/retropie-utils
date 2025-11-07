package storage

import (
	"context"
	"time"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"
)

type (
	RetrieveFileRequest struct {
		ToRetrieve  *fs.File
		Destination *fs.File
	}

	Storage interface {
		Init(ctx context.Context) error
		Retrieve(ctx context.Context, request RetrieveFileRequest) (*fs.File, error)
		Store(ctx context.Context, remoteDir string, file *fs.File) error
		StoreAll(ctx context.Context, remoteDir string, files []*fs.File) error
		GetFileLastModified(ctx context.Context, remoteDir string, file *fs.File) (*time.Time, error)
	}
)
