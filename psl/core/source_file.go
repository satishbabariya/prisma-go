// Package core provides core types and interfaces for PSL.
package core

// SourceFile represents a source file with its content.
type SourceFile struct {
	Path string
	Data string
}

// NewSourceFile creates a new SourceFile.
func NewSourceFile(path, data string) SourceFile {
	return SourceFile{
		Path: path,
		Data: data,
	}
}

