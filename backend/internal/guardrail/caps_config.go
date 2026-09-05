// Package guardrail implements the Policy Guardrail Agent (MRD 4.4, PRD
// FR4) — the "bounded and gated" core. Every StrategyDecision passes through
// Check before Execution ever runs.
package guardrail

// Caps is the fixed-cap config table (per user instruction: a config table,
// not hardcoded constants scattered through the rule logic). Static for the
// MVP, but centralized here so caps can move to a DB-backed config table
// later without touching guardrail.go's rule logic.
type Caps struct {
	MaxContactAttempts int
	MaxDiscountPct     float64
	ContactWindowStart int // hour, 0-23, simulated clock, inclusive
	ContactWindowEnd   int // hour, 0-23, simulated clock, exclusive
}

// DefaultCaps are the caps stated in MRD 4.4 / PRD FR4.
var DefaultCaps = Caps{
	MaxContactAttempts: 3,
	MaxDiscountPct:     10.0,
	ContactWindowStart: 9,
	ContactWindowEnd:   20,
}
