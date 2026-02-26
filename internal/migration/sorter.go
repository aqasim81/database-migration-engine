package migration

import "sort"

// Sort returns a new slice of migrations sorted by Version in lexicographic order.
// The sort is stable to preserve insertion order for equal versions.
func Sort(migrations []Migration) []Migration {
	sorted := make([]Migration, len(migrations))
	copy(sorted, migrations)

	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Version < sorted[j].Version
	})

	return sorted
}
