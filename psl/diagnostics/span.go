// Package diagnostics provides error and warning handling for PSL parsing and validation.
package diagnostics

// FileID represents the stable identifier for a PSL file.
type FileID uint32

const (
	// FileIDZero represents an empty or default file ID.
	FileIDZero FileID = 0
	// FileIDMax represents the maximum possible file ID.
	FileIDMax FileID = ^FileID(0)
)

// Span represents a location in a datamodel's text representation.
type Span struct {
	Start  int    `json:"start"`
	End    int    `json:"end"`
	FileID FileID `json:"file_id"`
}

// NewSpan creates a new span with the given parameters.
func NewSpan(start, end int, fileID FileID) Span {
	return Span{
		Start:  start,
		End:    end,
		FileID: fileID,
	}
}

// EmptySpan creates a new empty span.
func EmptySpan() Span {
	return Span{
		Start:  0,
		End:    0,
		FileID: FileIDZero,
	}
}

// Contains checks if the given position is inside the span (boundaries included).
func (s Span) Contains(position int) bool {
	return position >= s.Start && position <= s.End
}

// Overlaps checks if the given span overlaps with the current span.
func (s Span) Overlaps(other Span) bool {
	return s.FileID == other.FileID && (s.Contains(other.Start) || s.Contains(other.End))
}
