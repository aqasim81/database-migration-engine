package tracker_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aqasim81/database-migration-engine/internal/tracker"
)

func TestNew_returnsNonNil(t *testing.T) {
	t.Parallel()

	// nil pool is accepted at construction time; errors surface on use.
	tr := tracker.New(nil)
	assert.NotNil(t, tr)
}
