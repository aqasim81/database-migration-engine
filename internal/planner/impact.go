package planner

import "github.com/aqasim81/database-migration-engine/internal/analyzer"

// Size thresholds for impact estimation.
const (
	sizeUnknown int64 = -1
	size100MB   int64 = 100 * 1024 * 1024
	size1GB     int64 = 1024 * 1024 * 1024
)

// Duration estimate labels.
const (
	DurationUnder1s   = "< 1s"
	Duration1to10s    = "1-10s"
	Duration10sTo5min = "10s-5min"
	DurationOver5min  = "> 5min"
)

// Impact describes the estimated operational impact of a single finding.
type Impact struct {
	EstimatedLockDuration string
	LockType              string
	AffectedTable         string
	RequiresFullRewrite   bool
	RequiresFullScan      bool
	CanRunConcurrently    bool
	TableSizeBytes        int64 // -1 if unknown
}

// EstimateImpact returns the estimated impact of a single analyzer finding.
// tableSize is the table size in bytes; -1 if unknown.
func EstimateImpact(finding *analyzer.Finding, tableSize int64) Impact {
	desc := lookupRuleDescriptor(finding.Rule)

	lockType := finding.LockType
	if lockType == "" {
		lockType = desc.defaultLockType
	}

	return Impact{
		EstimatedLockDuration: estimateDuration(desc, tableSize),
		LockType:              lockType,
		AffectedTable:         finding.Table,
		RequiresFullRewrite:   desc.requiresFullRewrite,
		RequiresFullScan:      desc.requiresFullScan,
		CanRunConcurrently:    desc.canRunConcurrently,
		TableSizeBytes:        tableSize,
	}
}

type ruleDescriptor struct {
	defaultLockType     string
	requiresFullRewrite bool
	requiresFullScan    bool
	canRunConcurrently  bool
	durationFn          func(tableSize int64) string
}

//nolint:gochecknoglobals // package-level lookup table for rule impact mapping
var ruleDescriptors = map[string]ruleDescriptor{
	"create-index-not-concurrent": {
		defaultLockType: "SHARE",
		durationFn:      estimateBySizeIndex,
	},
	"add-column-volatile-default": {
		defaultLockType:     "ACCESS EXCLUSIVE",
		requiresFullRewrite: true,
		durationFn:          estimateBySizeRewrite,
	},
	"add-constraint-without-not-valid": {
		defaultLockType:  "ACCESS EXCLUSIVE",
		requiresFullScan: true,
		durationFn:       estimateBySizeScan,
	},
	"alter-column-type": {
		defaultLockType:     "ACCESS EXCLUSIVE",
		requiresFullRewrite: true,
		durationFn:          func(_ int64) string { return Duration10sTo5min },
	},
	"set-not-null": {
		defaultLockType:  "ACCESS EXCLUSIVE",
		requiresFullScan: true,
		durationFn:       estimateBySizeScan,
	},
	"drop-table": {
		defaultLockType: "ACCESS EXCLUSIVE",
		durationFn:      func(_ int64) string { return DurationUnder1s },
	},
	"vacuum-full": {
		defaultLockType:     "ACCESS EXCLUSIVE",
		requiresFullRewrite: true,
		durationFn:          estimateBySizeRewrite,
	},
	"lock-table": {
		defaultLockType: "EXPLICIT",
		durationFn:      func(_ int64) string { return Duration1to10s },
	},
	"rename": {
		defaultLockType: "ACCESS EXCLUSIVE",
		durationFn:      func(_ int64) string { return DurationUnder1s },
	},
}

var defaultDescriptor = ruleDescriptor{ //nolint:gochecknoglobals // fallback for unknown rules
	durationFn: func(_ int64) string { return DurationUnder1s },
}

func lookupRuleDescriptor(ruleID string) ruleDescriptor {
	if desc, ok := ruleDescriptors[ruleID]; ok {
		return desc
	}

	return defaultDescriptor
}

func estimateDuration(desc ruleDescriptor, tableSize int64) string {
	return desc.durationFn(tableSize)
}

// estimateBySizeIndex estimates duration for index creation (SHARE lock).
func estimateBySizeIndex(tableSize int64) string {
	switch {
	case tableSize == sizeUnknown:
		return Duration10sTo5min
	case tableSize < size100MB:
		return Duration1to10s
	case tableSize < size1GB:
		return Duration10sTo5min
	default:
		return DurationOver5min
	}
}

// estimateBySizeRewrite estimates duration for full table rewrite operations.
func estimateBySizeRewrite(tableSize int64) string {
	switch {
	case tableSize == sizeUnknown:
		return Duration10sTo5min
	case tableSize < size100MB:
		return Duration1to10s
	case tableSize < size1GB:
		return Duration10sTo5min
	default:
		return DurationOver5min
	}
}

// estimateBySizeScan estimates duration for full table scan operations.
func estimateBySizeScan(tableSize int64) string {
	switch {
	case tableSize == sizeUnknown:
		return Duration1to10s
	case tableSize < size100MB:
		return DurationUnder1s
	case tableSize < size1GB:
		return Duration1to10s
	default:
		return Duration10sTo5min
	}
}
