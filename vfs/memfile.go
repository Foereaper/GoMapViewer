package vfs

import (
	"bytes"
	"io/fs"
	"time"
)

// memFile implements fs.File for in-memory data.
type memFile struct {
	r    *bytes.Reader
	size int64
	name string
}

func (f *memFile) Read(p []byte) (int, error) {
	return f.r.Read(p)
}

func (f *memFile) Close() error {
	return nil
}

func (f *memFile) Stat() (fs.FileInfo, error) {
	return fileInfo{
		name: f.name,
		size: f.size,
	}, nil
}

type fileInfo struct {
	name string
	size int64
}

func (fi fileInfo) Name() string       { return fi.name }
func (fi fileInfo) Size() int64        { return fi.size }
func (fi fileInfo) Mode() fs.FileMode  { return 0444 }
func (fi fileInfo) ModTime() time.Time { return time.Time{} }
func (fi fileInfo) IsDir() bool        { return false }
func (fi fileInfo) Sys() interface{}   { return nil }
