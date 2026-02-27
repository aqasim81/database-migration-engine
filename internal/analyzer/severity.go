package analyzer

// Severity represents the danger level of a finding.
type Severity int

const (
	// Safe indicates no danger detected.
	Safe Severity = iota
	// Low indicates a minor concern.
	Low
	// Medium indicates moderate risk with workarounds available.
	Medium
	// High indicates significant risk — table lock or rewrite likely.
	High
	// Critical indicates data loss or extended downtime guaranteed.
	Critical
)

// String returns the uppercase label for the severity level.
func (s Severity) String() string {
	switch s {
	case Safe:
		return "SAFE"
	case Low:
		return "LOW"
	case Medium:
		return "MEDIUM"
	case High:
		return "HIGH"
	case Critical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Color returns an ANSI color code for terminal output.
func (s Severity) Color() string {
	switch s {
	case Safe:
		return "\033[32m" // green
	case Low:
		return "\033[36m" // cyan
	case Medium:
		return "\033[33m" // yellow
	case High:
		return "\033[31m" // red
	case Critical:
		return "\033[91m" // bright red
	default:
		return "\033[0m" // reset
	}
}
