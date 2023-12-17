package fs

import (
	"context"
	"os"
	"path/filepath"

	"github.com/TrevorEdris/retropie-utils/pkg/log"
	"github.com/rotisserie/eris"
)

type (
	Directory interface {
		GetName() string
		GetAbsolutePath() string
		GetAllFiles() []*File
		GetMatchingFiles(filetype FileType) ([]*File, error)
		RepopulateFiles(ctx context.Context) error
	}

	directory struct {
		Name     string
		Absolute string
		Files    []*File
	}
)

func NewDirectory(ctx context.Context, absolute string) (Directory, error) {
	d := &directory{
		Absolute: absolute,
		Name:     filepath.Base(absolute),
	}
	err := d.RepopulateFiles(ctx)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (d *directory) GetName() string {
	return d.Name
}

func (d *directory) GetAbsolutePath() string {
	return d.Absolute
}

func (d *directory) GetAllFiles() []*File {
	return d.Files
}

func (d *directory) GetMatchingFiles(filetype FileType) ([]*File, error) {
	matching := make([]*File, 0)
	for _, f := range d.Files {
		if f.FileType == filetype {
			matching = append(matching, f)
		}
	}

	return matching, nil
}

func (d *directory) RepopulateFiles(ctx context.Context) error {
	files := make([]*File, 0)
	err := filepath.Walk(d.Absolute, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, NewFile(
				path,
				info.ModTime(),
			))
		} else {
			log.FromCtx(ctx).Sugar().Debugf("Found sub-directory %s", info.Name())
		}
		return nil
	})
	if err != nil {
		return eris.Wrapf(err, "failed to repopulate files for directory %s", d.Name)
	}
	d.Files = files

	return nil
}
