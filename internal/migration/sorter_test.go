package migration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aqasim81/database-migration-engine/internal/migration"
)

func makeMigrations(t *testing.T, versions ...string) []migration.Migration {
	t.Helper()

	ms := make([]migration.Migration, len(versions))
	for i, v := range versions {
		ms[i] = migration.Migration{Version: v, Name: "test"}
	}

	return ms
}

func versions(t *testing.T, ms []migration.Migration) []string {
	t.Helper()

	vs := make([]string, len(ms))
	for i, m := range ms {
		vs[i] = m.Version
	}

	return vs
}

func TestSort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "already sorted stays sorted",
			input:    []string{"001", "002", "003"},
			expected: []string{"001", "002", "003"},
		},
		{
			name:     "reverse order is corrected",
			input:    []string{"003", "002", "001"},
			expected: []string{"001", "002", "003"},
		},
		{
			name:     "shuffled order is corrected",
			input:    []string{"002", "003", "001"},
			expected: []string{"001", "002", "003"},
		},
		{
			name:     "timestamp versions sort correctly",
			input:    []string{"20240201120000", "20240101120000", "20240301120000"},
			expected: []string{"20240101120000", "20240201120000", "20240301120000"},
		},
		{
			name:     "empty slice returns empty",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single element",
			input:    []string{"001"},
			expected: []string{"001"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := makeMigrations(t, tt.input...)
			result := migration.Sort(input)

			assert.Equal(t, tt.expected, versions(t, result))
		})
	}
}

func TestSort_doesNotMutateOriginal(t *testing.T) {
	t.Parallel()

	input := makeMigrations(t, "003", "001", "002")
	original := make([]string, len(input))
	for i, m := range input {
		original[i] = m.Version
	}

	migration.Sort(input)

	assert.Equal(t, original, versions(t, input), "original slice should not be mutated")
}
