// Package promise implements the Promise-to-Pay Tracker (MRD 4.6, PRD FR6).
// Unlike diagnosis/strategy/guardrail/execution (all pure functions),
// this agent inherently needs DB access: knowing whether a case has already
// broken a promise before requires reading prior promises for it.
package promise

import (
	"context"
	"fmt"
	"time"

	"vasooli/internal/db"
	"vasooli/internal/execution"
)

// strikesToEscalate is PRD FR6's literal "2 strikes -> auto-escalate" cap.
const strikesToEscalate = 2

// seedStrikeLookback is how far before "today" the scripted synthetic prior
// strike is dated, purely for a plausible-looking promised_date in the demo.
const seedStrikeLookback = 7 * 24 * time.Hour

// PromiseResult reports what this call did, for the (later) Audit Agent to
// log — mirrors the Evidence pattern used by every other agent so far.
type PromiseResult struct {
	CaseID            string
	PromiseCreated    bool
	SeededPriorStrike bool
	StrikeCount       int
	Escalated         bool
	Evidence          []string
}

// RecordIfAny logs a broken promise for this cycle's outcome (if any) and
// auto-escalates the case once its total strike count reaches 2.
//
// Only execution.OutcomeBrokenPromise produces a promise record: the
// pipeline's single pass per transaction collapses "customer committed to a
// date" and "that date was missed" into one outcome, so there is no
// separate pending-promise (kept=NULL) state to log in this design.
//
// For the fixture's scripted "repeated_broken_promise" case specifically,
// this seeds one synthetic prior-strike promise (kept=false, dated before
// simulatedToday) the first time it's called for that case, before
// recording this cycle's live broken promise — reaching the 2-strike
// escalation threshold from a single pipeline pass, per explicit approval.
// scenarios is the map from execution.LoadScenarios (transaction ID ->
// scenario tag); reused rather than re-implemented here.
func RecordIfAny(ctx context.Context, q db.PromiseQueries, outcome execution.ExecutionOutcome, transactionID string, caseID string, simulatedToday time.Time, scenarios map[string]string) (PromiseResult, error) {
	result := PromiseResult{CaseID: caseID}

	if outcome.Result != execution.OutcomeBrokenPromise {
		result.Evidence = []string{fmt.Sprintf("execution outcome=%s, not broken_promise; no promise recorded", outcome.Result)}
		return result, nil
	}

	if scenario, tagged := scenarios[transactionID]; tagged && scenario == execution.ScenarioRepeatedBrokenPromise {
		priorCount, err := q.CountBrokenPromises(ctx, caseID)
		if err != nil {
			return result, fmt.Errorf("promise: counting prior strikes: %w", err)
		}
		if priorCount == 0 {
			seedDate := simulatedToday.Add(-seedStrikeLookback)
			if err := q.InsertPromise(ctx, caseID, seedDate, false); err != nil {
				return result, fmt.Errorf("promise: seeding scripted prior strike: %w", err)
			}
			result.SeededPriorStrike = true
			result.Evidence = append(result.Evidence, fmt.Sprintf(
				"scenario=%s: seeded synthetic prior strike dated %s", scenario, seedDate.Format("2006-01-02")))
		}
	}

	if err := q.InsertPromise(ctx, caseID, simulatedToday, false); err != nil {
		return result, fmt.Errorf("promise: recording broken promise: %w", err)
	}
	result.PromiseCreated = true

	strikeCount, err := q.CountBrokenPromises(ctx, caseID)
	if err != nil {
		return result, fmt.Errorf("promise: recounting strikes: %w", err)
	}
	result.StrikeCount = strikeCount

	if strikeCount >= strikesToEscalate {
		if err := q.EscalateCase(ctx, caseID); err != nil {
			return result, fmt.Errorf("promise: escalating case: %w", err)
		}
		result.Escalated = true
		result.Evidence = append(result.Evidence, fmt.Sprintf("strikeCount=%d >= threshold=%d; auto-escalated", strikeCount, strikesToEscalate))
	} else {
		result.Evidence = append(result.Evidence, fmt.Sprintf("strikeCount=%d < threshold=%d; not yet escalated", strikeCount, strikesToEscalate))
	}

	return result, nil
}
