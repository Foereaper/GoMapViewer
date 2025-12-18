package vfs

import (
	"bytes"
	"io/fs"
	"path"
	"strings"
)

// FileSourceProvider allows querying the source MPQ for a file.
type FileSourceProvider interface {
	SourceOf(path string) (*FileSource, bool)
}

// FS exposes MPQStack as an io/fs filesystem.
type FS struct {
	stack *MPQStack
}

// NewFS creates a filesystem view of an MPQStack.
func NewFS(stack *MPQStack) *FS {
	return &FS{stack: stack}
}

// Open implements fs.FS.
func (f *FS) Open(name string) (fs.File, error) {
	// fs.FS paths are slash-separated and may start with '/'
	mpqPath := strings.TrimPrefix(name, "/")
	mpqPath = strings.ReplaceAll(mpqPath, "/", "\\")

	data, err := f.stack.ReadFile(mpqPath)
	if err != nil {
		return nil, fs.ErrNotExist
	}

	return &memFile{
		r:    bytes.NewReader(data),
		size: int64(len(data)),
		name: path.Base(mpqPath),
	}, nil
}

// SourceOf reports which MPQ supplies the given file.
func (f *FS) SourceOf(name string) (*FileSource, bool) {
	return f.stack.SourceOf(name)
}