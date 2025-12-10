// Package parserdatabase provides generator constants for Prisma ID generators.
package database

// UUIDSupportedVersions contains the versions of the uuid() ID generator supported by Prisma.
var UUIDSupportedVersions = []uint8{4, 7}

// CUIDSupportedVersions contains the versions of the cuid() ID generator supported by Prisma.
var CUIDSupportedVersions = []uint8{1, 2}

// DefaultUUIDVersion is the default version of the uuid() ID generator.
const DefaultUUIDVersion = 4

// DefaultCUIDVersion is the default version of the cuid() ID generator.
// Note: if you change this, you'll likely need to adapt existing tests that rely on cuid() sequences being already sorted
// (e.g., cuid(1), the current default, generates monotonically increasing sequences, cuid(2) doesn't).
const DefaultCUIDVersion = 1
