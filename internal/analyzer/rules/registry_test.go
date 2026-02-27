package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aqasim81/database-migration-engine/internal/analyzer/rules"
)

func TestNewDefaultRegistry_registersAllRules(t *testing.T) {
	t.Parallel()

	r := rules.NewDefaultRegistry()
	require.NotNil(t, r)
	assert.Len(t, r.Rules(), 9)
}

func TestNewDefaultRegistry_uniqueIDs(t *testing.T) {
	t.Parallel()

	r := rules.NewDefaultRegistry()
	seen := make(map[string]bool)

	for _, rule := range r.Rules() {
		id := rule.ID()
		assert.False(t, seen[id], "duplicate rule ID: %s", id)
		seen[id] = true
	}
}
