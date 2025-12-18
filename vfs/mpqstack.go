package vfs

import (
	"errors"
	"strings"

	"wowmap/mpq"
)

type FileSource struct {
	Archive *mpq.MPQ
	Path    string
	Order   int
}

// MPQStack represents a layered MPQ filesystem.
// Later-added archives override earlier ones.
type MPQStack struct {
	archives []*mpq.MPQ
	paths    []string
	loadOrder int
}

// New creates an empty MPQ stack.
func New() *MPQStack {
	return &MPQStack{}
}

// Add inserts an MPQ into the stack.
func (s *MPQStack) Add(a *mpq.MPQ) error {
	s.loadOrder++
	s.archives = append(s.archives, a)

	if p, ok := any(a).(interface{ Path() string }); ok {
		s.paths = append(s.paths, p.Path())
	} else {
		s.paths = append(s.paths, "")
	}

	return nil
}

// ReadFile reads the highest-priority version of a file.
func (s *MPQStack) ReadFile(name string) ([]byte, error) {
	if len(s.archives) == 0 {
		return nil, errors.New("no MPQs loaded")
	}

    // Search newest to oldest
	mpqPath := strings.ReplaceAll(name, "/", "\\")
	for i := len(s.archives) - 1; i >= 0; i-- {
		data, err := s.archives[i].ReadFile(mpqPath)
		if err == nil {
			return data, nil
		}
	}

	return nil, errors.New("file not found")
}

// HasFile checks if a file exists in any MPQ.
func (s *MPQStack) HasFile(name string) bool {
	mpqPath := strings.ReplaceAll(name, "/", "\\")
	for i := len(s.archives) - 1; i >= 0; i-- {
		if _, err := s.archives[i].ReadFile(mpqPath); err == nil {
			return true
		}
	}
	return false
}

// SourceOf returns which MPQ supplies a file.
func (s *MPQStack) SourceOf(name string) (*FileSource, bool) {
	mpqPath := strings.ReplaceAll(name, "/", "\\")

	for i := len(s.archives) - 1; i >= 0; i-- {
		if _, err := s.archives[i].ReadFile(mpqPath); err == nil {
			return &FileSource{
				Archive: s.archives[i],
				Path:    s.paths[i],
				Order:   i + 1,
			}, true
		}
	}

	return nil, false
}