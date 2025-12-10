// Package parserdatabase provides a simple string interner to reduce memory usage
// and allocation pressure of ParserDatabase.
//
// The StringIds returned by Intern are only valid for this specific instance
// of the interner they were interned with.
package database

import (
	"sync"
)

// StringId represents an interned string identifier.
type StringId uint32

// StringInterner provides string interning functionality.
type StringInterner struct {
	mu      sync.RWMutex
	strings []string
	index   map[string]StringId
}

// NewStringInterner creates a new StringInterner.
func NewStringInterner() *StringInterner {
	return &StringInterner{
		strings: make([]string, 0),
		index:   make(map[string]StringId),
	}
}

// Get returns the string for the given StringId, or empty string if not found.
func (si *StringInterner) Get(id StringId) string {
	si.mu.RLock()
	defer si.mu.RUnlock()

	if int(id) < len(si.strings) {
		return si.strings[id]
	}
	return ""
}

// Lookup returns the StringId for an already-interned string, or false if not found.
func (si *StringInterner) Lookup(s string) (StringId, bool) {
	si.mu.RLock()
	defer si.mu.RUnlock()

	id, ok := si.index[s]
	return id, ok
}

// Intern interns a string and returns its StringId.
// If the string is already interned, returns the existing StringId.
func (si *StringInterner) Intern(s string) StringId {
	si.mu.Lock()
	defer si.mu.Unlock()

	if id, ok := si.index[s]; ok {
		return id
	}

	id := StringId(len(si.strings))
	si.strings = append(si.strings, s)
	si.index[s] = id
	return id
}
