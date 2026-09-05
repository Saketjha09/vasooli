package guardrail

import (
	"fmt"

	"vasooli/internal/detector"
	"vasooli/internal/strategy"
)

const (
	RuleDisputeFlag        = "dispute_flag"
	RuleDoNotContact       = "do_not_contact"
	RuleMaxContactAttempts = "max_contact_attempts"
	RuleMaxDiscountPct     = "max_discount_pct"
	RuleContactHours       = "contact_hours"
)

// GuardrailResult extends the tech-arch struct contract (CaseID, Allowed,
// FinalTier, RuleFired, Reason) with:
//   - Evidence: every rule checked and its outcome, even when nothing fired,
//     so e.g. a compliant discount still shows "discountPct=8.0 <= cap=10.0"
//     in the audit trail — not just the rule that blocked/held.
//   - Held: true only for the contact-hours rule. PRD FR4 requires that case
//     to be held, not canceled/escalated, which is a different outcome from
//     every other blocking rule (which per MRD 4.4 escalates).
//
// verified is unexported and set only by Check (never accessible in a
// struct literal from another package), so execution.Simulate can require
// proof a result actually came from a real Check call rather than a
// hand-built fake — closing the "no case reaches Execution without a
// completed Guardrail check" rule at compile time, not just by convention.
type GuardrailResult struct {
	CaseID    string
	Allowed   bool
	FinalTier string
	RuleFired string
	Reason    string
	Held      bool
	Evidence  []string
	verified  bool
}

// Verified reports whether this result was produced by a real call to
// Check. Always true for any GuardrailResult obtained that way.
func (g GuardrailResult) Verified() bool {
	return g.verified
}

// Check runs a StrategyDecision through the fixed caps, in order, first
// blocking/holding match wins. No case reaches Execution without passing
// through here first.
//
// event carries the customer context (disputed_flag, do_not_contact) that
// detector.Event already has — reused directly rather than introducing a
// separate CustomerContext type, since it's the same data.
func Check(decision strategy.StrategyDecision, event detector.Event, caps Caps, simulatedHour int, priorContactAttempts int) GuardrailResult {
	result := evaluate(decision, event, caps, simulatedHour, priorContactAttempts)
	result.verified = true
	return result
}

// evaluate holds the actual rule logic. Never exported directly — always
// go through Check so the verified marker gets set.
func evaluate(decision strategy.StrategyDecision, event detector.Event, caps Caps, simulatedHour int, priorContactAttempts int) GuardrailResult {
	result := GuardrailResult{
		CaseID:    decision.CaseID,
		Allowed:   true,
		FinalTier: decision.Tier,
	}

	requiresContact := decision.Channel != strategy.ChannelNone

	// Rule 1: dispute flag — checked first and unconditionally, per MRD 4.4:
	// detected *before* any recovery action fires, regardless of tier chosen.
	disputeEvidence := fmt.Sprintf("customer.disputed_flag=%t", event.DisputedFlag)
	result.Evidence = append(result.Evidence, disputeEvidence)
	if event.DisputedFlag {
		result.Allowed = false
		result.FinalTier = strategy.TierEscalate
		result.RuleFired = RuleDisputeFlag
		result.Reason = disputeEvidence + "; auto-lock and escalate, no contact"
		return result
	}

	// Rule 2: do-not-contact — hard block, escalate (MRD 4.4: "if blocked,
	// ... the case is escalated").
	dncEvidence := fmt.Sprintf("customer.do_not_contact=%t", event.DoNotContact)
	result.Evidence = append(result.Evidence, dncEvidence)
	if event.DoNotContact {
		result.Allowed = false
		result.FinalTier = strategy.TierEscalate
		result.RuleFired = RuleDoNotContact
		result.Reason = dncEvidence + "; hard block, escalate"
		return result
	}

	// Rule 3: max contact attempts — only meaningful for tiers that contact
	// the customer at all.
	if requiresContact {
		attemptsEvidence := fmt.Sprintf("priorContactAttempts=%d, cap=%d", priorContactAttempts, caps.MaxContactAttempts)
		result.Evidence = append(result.Evidence, attemptsEvidence)
		if priorContactAttempts >= caps.MaxContactAttempts {
			result.Allowed = false
			result.FinalTier = strategy.TierEscalate
			result.RuleFired = RuleMaxContactAttempts
			result.Reason = attemptsEvidence + "; cap reached, escalate"
			return result
		}
	}

	// Rule 4: max discount — checked against the real DiscountPct Strategy
	// computed, not assumed compliant.
	if decision.DiscountPct > 0 {
		discountEvidence := fmt.Sprintf("discountPct=%.1f, cap=%.1f", decision.DiscountPct, caps.MaxDiscountPct)
		if decision.DiscountPct > caps.MaxDiscountPct {
			result.Evidence = append(result.Evidence, discountEvidence+" (exceeds cap)")
			result.Allowed = false
			result.FinalTier = strategy.TierEscalate
			result.RuleFired = RuleMaxDiscountPct
			result.Reason = discountEvidence + "; exceeds cap, escalate"
			return result
		}
		result.Evidence = append(result.Evidence, discountEvidence+" (within cap)")
	}

	// Rule 5: contact hours — hold, don't cancel. Simulated clock only.
	if requiresContact {
		hoursEvidence := fmt.Sprintf("simulatedHour=%d, window=[%d,%d)", simulatedHour, caps.ContactWindowStart, caps.ContactWindowEnd)
		result.Evidence = append(result.Evidence, hoursEvidence)
		if simulatedHour < caps.ContactWindowStart || simulatedHour >= caps.ContactWindowEnd {
			result.Allowed = false
			result.FinalTier = decision.Tier // NOT escalated — held for retry in-window
			result.RuleFired = RuleContactHours
			result.Reason = hoursEvidence + "; outside contact hours, held for retry"
			result.Held = true
			return result
		}
	}

	result.Reason = "all checks passed"
	return result
}
