package syncer

import (
	"context"
	"time"

	"github.com/TrevorEdris/retropie-utils/pkg/fs"
	"github.com/TrevorEdris/retropie-utils/pkg/log"
	"github.com/TrevorEdris/retropie-utils/pkg/storage"
	"github.com/rotisserie/eris"
	"go.uber.org/zap"
)

type (
	Syncer interface {
		Sync(ctx context.Context) error
	}

	syncer struct {
		cfg     Config
		storage storage.Storage
	}

	Schedule struct{}
)

const (
	// timeToDirFmt describes the folder structure for storing files
	// in a time-based format, such that the same file uploaded twice
	// but at separate hours would be stored in two separate locations.
	//
	// Example:
	// December 17, 2023 at 1:18pm EST
	// 2023/12/17/1
	timeToDirFmt = "2006/01/02/15"
)

func NewSyncer(ctx context.Context, cfg Config) (Syncer, error) {
	var storageClient storage.Storage
	var err error
	if cfg.Storage.S3.Enabled {
		storageClient, err = storage.NewS3Storage(ctx, cfg.Storage.S3)
	} else {
		err = eris.New("no storage clients enabled")
	}
	if err != nil {
		return nil, err
	}
	err = storageClient.Init(ctx)
	if err != nil {
		return nil, err
	}
	return &syncer{
		cfg:     cfg,
		storage: storageClient,
	}, nil
}

func (s *syncer) Sync(ctx context.Context) error {
	log.FromCtx(ctx).Info("Looking for roms in subfolders", zap.String("directory", s.cfg.RomsFolder))
	romDir, err := fs.NewDirectory(ctx, s.cfg.RomsFolder)
	if err != nil {
		return err
	}
	if len(romDir.GetAllFiles()) == 0 {
		log.FromCtx(ctx).Warn("No files found", zap.String("directory", s.cfg.RomsFolder))
	}
	remoteDir := time.Now().Format(timeToDirFmt)
	log.FromCtx(ctx).Info("Syncs enabled", zap.Bool("roms", s.cfg.Sync.Roms), zap.Bool("saves", s.cfg.Sync.Saves), zap.Bool("states", s.cfg.Sync.States))
	if s.cfg.Sync.Roms {
		log.FromCtx(ctx).Info("Syncing ROMs")
		err = s.sync(ctx, romDir, fs.Rom, remoteDir)
		if err != nil {
			return err
		}
	}
	if s.cfg.Sync.Saves {
		log.FromCtx(ctx).Info("Syncing saves")
		err = s.sync(ctx, romDir, fs.Save, remoteDir)
		if err != nil {
			return err
		}
	}
	if s.cfg.Sync.States {
		log.FromCtx(ctx).Info("Syncing states")
		err = s.sync(ctx, romDir, fs.State, remoteDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *syncer) sync(ctx context.Context, sourceDir fs.Directory, filetype fs.FileType, remoteDir string) error {
	files, err := sourceDir.GetMatchingFiles(filetype)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		log.FromCtx(ctx).Warn("No matching files")
		return nil
	}
	log.FromCtx(ctx).Sugar().Infof("Found %d matching files", len(files))
	err = s.storage.StoreAll(ctx, remoteDir, files)
	if err != nil {
		return err
	}
	return nil
}
