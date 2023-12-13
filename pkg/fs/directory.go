package fs

import (
	"os"
	"path/filepath"
)

type (
	Directory interface {
		GetName() string
		GetAbsolutePath() string
		GetMatchingFiles(filetype FileType) ([]*File, error)
		RepopulateFiles() error
	}

	directory struct {
		Name     string
		Absolute string
		Files    []*File
	}
)

func NewDirectory(absolute string) (Directory, error) {
	d := &directory{
		Absolute: absolute,
		Name:     filepath.Base(absolute),
	}
	err := d.RepopulateFiles()
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

func (d *directory) RepopulateFiles() error {
	files := make([]*File, 0)
	err := filepath.Walk(d.Absolute, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, NewFile(
				filepath.Join(path, info.Name()),
				info.ModTime(),
			))
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
