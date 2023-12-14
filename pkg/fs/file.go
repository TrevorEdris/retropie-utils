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
	Generic
)

var (
	suffixToFileType = map[string]FileType{
		// Roms
		"gb":  Rom,
		"gbc": Rom,
		"gba": Rom,
		"smc": Rom,
		"z64": Rom,
		"nes": Rom,
		// Saves
		"srm": Save,
		"sav": Save,
		"rtc": Save,
		// States
		"state":  State,
		"state1": State,
		"state2": State,
		"state3": State,
		"state4": State,
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
		Dir:          filepath.Dir(absolutePath),
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
	ft, ok := suffixToFileType[filepath.Ext(filename)]
	if !ok {
		return Generic
	}
	return ft
}
