package fs

import (
	"path/filepath"
	"time"
)

type FileType int

const (
	Rom FileType = iota
	Save
	State
	Other
)

var (
	suffixToFileType = map[string]FileType{
		// Roms
		".gb":  Rom,
		".gbc": Rom,
		".gba": Rom,
		".smc": Rom,
		".z64": Rom,
		".nes": Rom,
		// Saves
		".srm": Save,
		".sav": Save,
		".rtc": Save,
		// States
		".state":  State,
		".state1": State,
		".state2": State,
		".state3": State,
		".state4": State,
	}
)

type (
	File struct {
		Dir          string
		Absolute     string
		Name         string
		LastModified time.Time
		FileType     FileType
	}
)

func NewFile(absolutePath string, lastModified time.Time) *File {
	return &File{
		Dir:          filepath.Base(filepath.Dir(absolutePath)),
		Absolute:     absolutePath,
		Name:         filepath.Base(absolutePath),
		LastModified: lastModified,
		FileType:     parseFiletype(filepath.Base(absolutePath)),
	}
}

func (f *File) IsOlderThan(other *File) bool {
	return f.LastModified.Before(other.LastModified)
}

func parseFiletype(filename string) FileType {
	ext := filepath.Ext(filename)
	ft, ok := suffixToFileType[ext]
	if !ok {
		return Other
	}
	return ft
}
