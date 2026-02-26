package migration

import (
	"crypto/sha256"
	"encoding/hex"
)

// Migration represents a single database migration loaded from disk.
type Migration struct {
	Version  string // "001" or "20240101120000" — extracted from filename
	Name     string // "create_users" — extracted from filename
	UpSQL    string // Contents of the .up.sql file
	DownSQL  string // Contents of the .down.sql file (empty if none)
	Checksum string // SHA-256 hex digest of UpSQL
	FilePath string // Path to the .up.sql file
}

// ComputeChecksum returns the SHA-256 hex digest of the given SQL string.
func ComputeChecksum(sql string) string {
	h := sha256.Sum256([]byte(sql))

	return hex.EncodeToString(h[:])
}
