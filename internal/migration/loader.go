package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// filenamePattern matches migration files in two formats:
//
//	V{version}_{name}.up.sql   (e.g., V001_create_users.up.sql)
//	{timestamp}_{name}.up.sql  (e.g., 20240101120000_create_users.up.sql)
var filenamePattern = regexp.MustCompile( //nolint:gochecknoglobals // compiled once, used by LoadFromDir
	`^(?:V(\d+)|(\d{14}))_(.+)\.(up|down)\.sql$`,
)

// LoadFromDir scans a directory for migration files and returns them as unsorted Migration values.
// Files that do not match the expected naming pattern are skipped.
func LoadFromDir(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading migrations directory %s: %w", dir, err)
	}

	grouped := scanEntries(entries)

	return buildMigrations(grouped, dir)
}

// migrationFile is an intermediate struct for pairing up/down files.
type migrationFile struct {
	version  string
	name     string
	upFile   string // filename only (not full path)
	downFile string // filename only (not full path)
}

// scanEntries groups directory entries by version+name key.
func scanEntries(entries []os.DirEntry) map[string]*migrationFile {
	grouped := make(map[string]*migrationFile)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := filenamePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		version := matches[1] // V-prefixed version
		if version == "" {
			version = matches[2] // timestamp version
		}

		name := matches[3]
		direction := matches[4]
		key := version + "_" + name

		mf, ok := grouped[key]
		if !ok {
			mf = &migrationFile{version: version, name: name}
			grouped[key] = mf
		}

		if direction == "up" {
			mf.upFile = entry.Name()
		} else {
			mf.downFile = entry.Name()
		}
	}

	return grouped
}

// buildMigrations reads file contents and constructs Migration values from grouped files.
func buildMigrations(grouped map[string]*migrationFile, dir string) ([]Migration, error) {
	var migrations []Migration

	for _, mf := range grouped {
		if mf.upFile == "" {
			continue // orphan .down.sql — skip
		}

		m, err := readMigration(mf, dir)
		if err != nil {
			return nil, err
		}

		migrations = append(migrations, m)
	}

	return migrations, nil
}

// readMigration reads up/down SQL files and builds a Migration.
func readMigration(mf *migrationFile, dir string) (Migration, error) {
	upPath := filepath.Join(dir, mf.upFile)

	upData, err := os.ReadFile(upPath)
	if err != nil {
		return Migration{}, fmt.Errorf("reading migration file %s: %w", upPath, err)
	}

	upSQL := strings.TrimSpace(string(upData))

	var downSQL string

	if mf.downFile != "" {
		downPath := filepath.Join(dir, mf.downFile)

		downData, err := os.ReadFile(downPath)
		if err != nil {
			return Migration{}, fmt.Errorf("reading migration file %s: %w", downPath, err)
		}

		downSQL = strings.TrimSpace(string(downData))
	}

	return Migration{
		Version:  mf.version,
		Name:     mf.name,
		UpSQL:    upSQL,
		DownSQL:  downSQL,
		Checksum: ComputeChecksum(upSQL),
		FilePath: upPath,
	}, nil
}
