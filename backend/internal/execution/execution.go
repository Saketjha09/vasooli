// Package execution implements the Execution Agent (MRD 4.5, PRD FR5): it
// always runs (guardrail must run before it, never instead of it) and
// simulates whatever action guardrail's FinalTier actually authorizes — no
// real Razorpay calls, no fabricated recovered/no_response/broken_promise
// outcome for a case that was blocked or held.
package execution

import (
	"errors"
	"fmt"
	"math"

	"vasooli/internal/detector"
	"vasooli/internal/guardrail"
	"vasooli/internal/strategy"
)

// ErrGuardrailNotVerified is returned by Simulate when the GuardrailResult
// passed in didn't come from a real guardrail.Check call. Returned rather
// than a panic: a panic here with nothing recovering it would crash the
// whole server process on one bad case, taking down every other
// in-flight/subsequent request along with it. The caller (the pipeline
// orchestrator) is expected to catch this and record it as an
// internal-error outcome for that one case, not let it abort the batch run.
var ErrGuardrailNotVerified = errors.New("execution: GuardrailResult did not come from guardrail.Check")

const (
	OutcomeRecovered     = "recovered"
	OutcomeNoResponse    = "no_response"
	OutcomeBrokenPromise = "broken_promise"

	// OutcomeEscalated and OutcomeHeld are additions beyond the MRD's literal
	// 3-value outcome list (recovered/no_response/broken_promise) — flagged
	// since a blocked or held case has no automated-contact result to report
	// and fabricating one of the 3 would misrepresent what happened.
	OutcomeEscalated = "escalated"
	OutcomeHeld      = "held"
)

// ScenarioRepeatedBrokenPromise is the fixture-tag value (from
// backend/fixtures/demo_dataset.json) that scripts a deterministic
// broken_promise outcome for the demo's repeated-broken-promise showcase
// case, overriding the historyScore bucket below.
const ScenarioRepeatedBrokenPromise = "repeated_broken_promise"

// historyScore bucket thresholds for contacted tiers (nudge/incentivized_nudge).
// Named and logged in evidence, same pattern as strategy's threshold.
const (
	recoveredHistoryThreshold     = 70 // >= this: recovered
	brokenPromiseHistoryThreshold = 40 // < this: broken_promise; between the two: no_response
)

// ExecutionOutcome matches the tech-arch struct contract (CaseID, Result,
// Amount) with an added Evidence trail, consistent with every other agent
// in this pipeline.
type ExecutionOutcome struct {
	CaseID   string
	Result   string
	Amount   float64
	Evidence []string
}

// Simulate produces the synthetic outcome for one case. scenarios is the map
// loaded by LoadScenarios (transaction ID -> scenario tag); pass nil/empty
// if no fixture scenarios apply.
//
// Returns ErrGuardrailNotVerified if g wasn't produced by a real
// guardrail.Check call — see that error's doc comment for why this fails
// safely instead of panicking.
func Simulate(g guardrail.GuardrailResult, decision strategy.StrategyDecision, event detector.Event, scenarios map[string]string) (ExecutionOutcome, error) {
	if !g.Verified() {
		return ExecutionOutcome{}, ErrGuardrailNotVerified
	}

	out := ExecutionOutcome{CaseID: g.CaseID}

	switch {
	case g.Held:
		out.Result = OutcomeHeld
		out.Amount = 0
		out.Evidence = []string{fmt.Sprintf("guardrail held: rule=%s", g.RuleFired)}
		return out, nil

	case g.FinalTier == strategy.TierEscalate:
		out.Result = OutcomeEscalated
		out.Amount = 0
		if g.RuleFired != "" {
			out.Evidence = []string{fmt.Sprintf("guardrail blocked and escalated: rule=%s", g.RuleFired)}
		} else {
			out.Evidence = []string{"strategy selected escalate directly; no automated contact"}
		}
		return out, nil

	case g.FinalTier == strategy.TierSilentRetry:
		out.Result = OutcomeRecovered
		out.Amount = event.Amount
		out.Evidence = []string{"tier=silent_retry, transient failure auto-resolved"}
		return out, nil
	}

	// Remaining tiers (nudge, incentivized_nudge) involve customer contact.
	if scenario, ok := scenarios[event.TransactionID]; ok && scenario == ScenarioRepeatedBrokenPromise {
		out.Result = OutcomeBrokenPromise
		out.Amount = 0
		out.Evidence = []string{fmt.Sprintf("scenario=%s (scripted demo override)", ScenarioRepeatedBrokenPromise)}
		return out, nil
	}

	switch {
	case event.HistoryScore >= recoveredHistoryThreshold:
		out.Result = OutcomeRecovered
		out.Amount = applyDiscount(event.Amount, decision.DiscountPct)
		out.Evidence = []string{fmt.Sprintf("historyScore=%d >= recoveredThreshold=%d", event.HistoryScore, recoveredHistoryThreshold)}
	case event.HistoryScore < brokenPromiseHistoryThreshold:
		out.Result = OutcomeBrokenPromise
		out.Amount = 0
		out.Evidence = []string{fmt.Sprintf("historyScore=%d < brokenPromiseThreshold=%d", event.HistoryScore, brokenPromiseHistoryThreshold)}
	default:
		out.Result = OutcomeNoResponse
		out.Amount = 0
		out.Evidence = []string{fmt.Sprintf("historyScore=%d between brokenPromiseThreshold=%d and recoveredThreshold=%d", event.HistoryScore, brokenPromiseHistoryThreshold, recoveredHistoryThreshold)}
	}
	return out, nil
}

func applyDiscount(amount, discountPct float64) float64 {
	if discountPct <= 0 {
		return amount
	}
	return math.Round(amount*(1-discountPct/100)*100) / 100
}
