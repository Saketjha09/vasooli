// Package strategy implements the Strategy Agent (MRD 4.3, PRD FR3): maps
// (root cause, confidence, history) to exactly one of the 4 tiers plus a
// contact channel, via an inspectable table/switch — no scored model.
//
// SelectTier handles a case's *first* pass through the pipeline.
// NudgeIgnoredUpgrade (below) handles MRD tier 3's literal repeat-cycle
// trigger — "nudge ignored once -> incentivized nudge" — called by the
// pipeline orchestrator (component 8) in a second cycle over cases whose
// first-cycle tier was nudge and outcome was no_response.
package strategy

import (
	"fmt"

	"vasooli/internal/diagnosis"
)

const (
	TierSilentRetry       = "silent_retry"
	TierNudge             = "nudge"
	TierIncentivizedNudge = "incentivized_nudge"
	TierEscalate          = "escalate"

	ChannelNone  = ""
	ChannelSMS   = "sms"
	ChannelEmail = "email"
)

// willfulNonpaymentHistoryThreshold is the historyScore cutoff for a
// willful_nonpayment case: at or above it the customer gets one more
// (incentivized) chance; below it the case escalates straight to a human.
// Named and logged in evidence so the cutoff is never a bare inline number.
const willfulNonpaymentHistoryThreshold = 40

// incentivizedNudgeDiscountPct is the fixed discount offered on the
// incentivized_nudge tier — a constant, not computed per case, and set
// safely under guardrail's MaxDiscountPct cap (10%) so Guardrail has a real
// number to check rather than assuming this tier is always compliant.
const incentivizedNudgeDiscountPct = 8.0

// StrategyDecision extends the tech-arch struct contract with an Evidence
// trail and a DiscountPct (not in the original CaseID/Tier/Channel-only
// shape) so the tier choice is auditable and Guardrail can check the actual
// discount value against its cap — both added at the user's explicit
// request in this component's approval.
type StrategyDecision struct {
	CaseID      string
	Tier        string
	Channel     string
	DiscountPct float64
	Evidence    []string
	// Alternatives holds a genuine one-line "why not" reason for each of the
	// 3 tiers NOT chosen, derived from the same rule table SelectTier uses —
	// not the Evidence trail, and not a fabricated list. This is what the
	// MRD's "chosen action over the alternatives" explainability claim
	// literally requires for Strategy specifically.
	Alternatives []string
}

// SelectTier assigns exactly one tier + channel for a case's first pass
// through the pipeline, purely from root cause (+ historyScore for the
// willful_nonpayment split).
func SelectTier(d diagnosis.DiagnosisResult, historyScore int) StrategyDecision {
	base := StrategyDecision{CaseID: d.CaseID}

	switch d.RootCause {
	case diagnosis.RootCauseDisputed:
		base.Tier = TierEscalate
		base.Channel = ChannelNone
		base.Evidence = []string{"root_cause=disputed"}

	case diagnosis.RootCauseTransientGateway:
		base.Tier = TierSilentRetry
		base.Channel = ChannelNone
		base.Evidence = []string{"root_cause=transient_gateway"}

	case diagnosis.RootCauseCardExpired, diagnosis.RootCauseInsufficientFunds:
		base.Tier = TierNudge
		base.Channel = ChannelEmail
		base.Evidence = []string{fmt.Sprintf("root_cause=%s", d.RootCause)}

	case diagnosis.RootCauseCheckoutFriction:
		base.Tier = TierNudge
		base.Channel = ChannelSMS
		base.Evidence = []string{"root_cause=checkout_friction"}

	case diagnosis.RootCauseWillfulNonpayment:
		if historyScore >= willfulNonpaymentHistoryThreshold {
			base.Tier = TierIncentivizedNudge
			base.Channel = ChannelEmail
			base.DiscountPct = incentivizedNudgeDiscountPct
		} else {
			base.Tier = TierEscalate
			base.Channel = ChannelNone
		}
		base.Evidence = []string{
			"root_cause=willful_nonpayment",
			fmt.Sprintf("historyScore=%d, threshold=%d, meetsThreshold=%t",
				historyScore, willfulNonpaymentHistoryThreshold, historyScore >= willfulNonpaymentHistoryThreshold),
		}

	default:
		// Root causes are a closed set produced by diagnosis.Classify; an
		// unrecognized value here means diagnosis and strategy have drifted
		// out of sync, which must fail loudly rather than guess a tier.
		panic(fmt.Sprintf("strategy: no tier mapping for root cause %q", d.RootCause))
	}

	base.Alternatives = rejectedAlternatives(d.RootCause, historyScore, base.Tier)
	return base
}

// rejectedAlternatives returns one genuine "why not" line per tier other
// than chosenTier, derived from the exact same rule table SelectTier just
// applied above — not a copy of Evidence, and not invented after the fact.
func rejectedAlternatives(rootCause string, historyScore int, chosenTier string) []string {
	reasons := map[string]string{
		TierSilentRetry:       "not applicable — no automated retry alone resolves this root cause",
		TierNudge:             "not the right fit for this case",
		TierIncentivizedNudge: "not the right fit for this case",
		TierEscalate:          "not warranted yet",
	}

	switch rootCause {
	case diagnosis.RootCauseDisputed:
		reasons[TierSilentRetry] = "not applicable — disputed cases must not receive any automated action, silent or otherwise"
		reasons[TierNudge] = "not applicable — contacting a customer with an active dispute is compliance-unsafe"
		reasons[TierIncentivizedNudge] = "not applicable — same reason as nudge; disputed cases are locked from all contact tiers"

	case diagnosis.RootCauseTransientGateway:
		reasons[TierNudge] = "not needed — root cause is transient_gateway, no customer-side fix is required"
		reasons[TierIncentivizedNudge] = "not needed — same as nudge; no customer action applies to a transient failure"
		reasons[TierEscalate] = "not warranted — transient failures resolve without human intervention"

	case diagnosis.RootCauseCardExpired, diagnosis.RootCauseInsufficientFunds:
		reasons[TierSilentRetry] = fmt.Sprintf("not applicable — root cause is %s, not transient; retrying without customer action won't succeed", rootCause)
		reasons[TierIncentivizedNudge] = "not yet — no discount needed on a first attempt; reserved for a nudge that goes unanswered"
		reasons[TierEscalate] = "not warranted — this is a routine, customer-fixable cause, not yet requiring human intervention"

	case diagnosis.RootCauseCheckoutFriction:
		reasons[TierSilentRetry] = "not applicable — root cause is checkout_friction, not a transient gateway issue"
		reasons[TierIncentivizedNudge] = "not yet — no discount needed on a first attempt; reserved for a nudge that goes unanswered"
		reasons[TierEscalate] = "not warranted — this is a routine, customer-fixable cause, not yet requiring human intervention"

	case diagnosis.RootCauseWillfulNonpayment:
		reasons[TierSilentRetry] = "not applicable — root cause is willful_nonpayment, not transient"
		if historyScore >= willfulNonpaymentHistoryThreshold {
			reasons[TierNudge] = fmt.Sprintf("insufficient — historyScore=%d meets threshold=%d, so an incentivized offer is tried directly rather than a plain nudge", historyScore, willfulNonpaymentHistoryThreshold)
			reasons[TierEscalate] = fmt.Sprintf("not yet — historyScore=%d meets threshold=%d, so one incentivized attempt is made before escalating", historyScore, willfulNonpaymentHistoryThreshold)
		} else {
			reasons[TierNudge] = fmt.Sprintf("insufficient — historyScore=%d is below threshold=%d, a plain nudge has low likelihood of success", historyScore, willfulNonpaymentHistoryThreshold)
			reasons[TierIncentivizedNudge] = fmt.Sprintf("not offered — historyScore=%d is below threshold=%d; policy escalates rather than discounting for low-history willful nonpayment", historyScore, willfulNonpaymentHistoryThreshold)
		}
	}

	var out []string
	for _, tier := range []string{TierSilentRetry, TierNudge, TierIncentivizedNudge, TierEscalate} {
		if tier == chosenTier {
			continue
		}
		out = append(out, fmt.Sprintf("%s: %s", tier, reasons[tier]))
	}
	return out
}

// NudgeIgnoredUpgrade builds the tier-3 upgrade decision for a case whose
// first cycle was tier=nudge with a no_response outcome — MRD's literal
// "nudge ignored once -> incentivized nudge" trigger. The caller (the
// pipeline orchestrator) is responsible for checking that first-cycle
// condition before calling this; it unconditionally builds the upgrade.
func NudgeIgnoredUpgrade(caseID string) StrategyDecision {
	return StrategyDecision{
		CaseID:      caseID,
		Tier:        TierIncentivizedNudge,
		Channel:     ChannelEmail,
		DiscountPct: incentivizedNudgeDiscountPct,
		Evidence:    []string{"prior cycle: tier=nudge, outcome=no_response; upgrading per MRD tier 3 trigger"},
		Alternatives: []string{
			fmt.Sprintf("%s: not applicable — this case already required customer contact", TierSilentRetry),
			fmt.Sprintf("%s: already tried once and ignored; escalating the offer rather than repeating it unchanged", TierNudge),
			fmt.Sprintf("%s: not yet — one incentivized attempt is tried before escalating to a human", TierEscalate),
		},
	}
}
