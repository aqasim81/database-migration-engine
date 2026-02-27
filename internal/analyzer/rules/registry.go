package rules

import "github.com/aqasim81/database-migration-engine/internal/analyzer"

// NewDefaultRegistry returns a Registry with all built-in detection rules.
func NewDefaultRegistry() *analyzer.Registry {
	r := analyzer.NewRegistry()
	r.Register(NewCreateIndexRule())
	r.Register(NewAddColumnRule())
	r.Register(NewAddConstraintRule())
	r.Register(NewAlterColumnTypeRule())
	r.Register(NewSetNotNullRule())
	r.Register(NewDropTableRule())
	r.Register(NewVacuumFullRule())
	r.Register(NewLockTableRule())
	r.Register(NewRenameRule())

	return r
}
