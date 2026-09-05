// Package pipeline implements the orchestrator (architecture doc section
// 2): Detector -> Diagnosis -> Strategy -> Guardrail -> Execution -> Promise
// -> Audit, in that order, per case, plus a second cycle for MRD tier 3's
// "nudge ignored once -> incentivized_nudge" upgrade.
package pipeline

import (
	"context"
	"fmt"
	"time"

	"vasooli/internal/audit"
	"vasooli/internal/db"
	"vasooli/internal/detector"
	"vasooli/internal/diagnosis"
	"vasooli/internal/execution"
	"vasooli/internal/guardrail"
	"vasooli/internal/promise"
	"vasooli/internal/strategy"
)

// Config holds the run-level settings for one batch run — all simulated,
// per MRD's "simulated clock" scope.
type Config struct {
	Caps           guardrail.Caps
	SimulatedHour  int       // 0-23, drives guardrail's contact-hours rule
	SimulatedToday time.Time // drives promise.RecordIfAny's promised_date values
	ScenariosPath  string    // path to backend/fixtures/demo_dataset.json
}

// Queries is everything the orchestrator needs from the DB — the union of
// every stage's own query interface, all satisfied by *db.DB.
type Queries interface {
	db.Queries
	db.PromiseQueries
	db.AuditQueries
	db.CaseQueries
}

// Summary is a convenience rollup of one batch run, returned to the caller
// (e.g. the API layer) without requiring a separate query.
type Summary struct {
	TotalCases     int
	TierBreakdown  map[string]int // keyed by the FINAL tier after all cycles
	MoneyRecovered float64
}

// caseState carries one transaction's running state across cycles within a
// single RunBatch call — not persisted itself; the DB rows are the
// persisted record.
type caseState struct {
	event       detector.Event
	caseID      string
	diagResult  diagnosis.DiagnosisResult
	lastTier    string
	lastOutcome string
	lastAmount  float64
	attempts    int // count of cycles where a contact tier was actually allowed (not held/blocked) — feeds guardrail's priorContactAttempts
}

// RunBatch resets derived state, then runs the full synthetic batch through
// the pipeline. Safe to call repeatedly: each call starts from the same
// clean slate (customers/transactions are untouched seed data), so results
// are identical across runs.
func RunBatch(ctx context.Context, q Queries, cfg Config) (Summary, error) {
	if err := q.ResetDerivedState(ctx); err != nil {
		return Summary{}, fmt.Errorf("pipeline: resetting derived state: %w", err)
	}

	scenarios, err := execution.LoadScenarios(cfg.ScenariosPath)
	if err != nil {
		return Summary{}, fmt.Errorf("pipeline: loading scenarios: %w", err)
	}

	events, err := detector.IngestBatch(ctx, q)
	if err != nil {
		return Summary{}, fmt.Errorf("pipeline: ingesting batch: %w", err)
	}

	states := make([]*caseState, 0, len(events))
	for _, event := range events {
		caseID, err := q.CreateCase(ctx, event.TransactionID)
		if err != nil {
			return Summary{}, fmt.Errorf("pipeline: creating case for transaction %s: %w", event.TransactionID, err)
		}
		states = append(states, &caseState{event: event, caseID: caseID})
	}

	for _, st := range states {
		if err := runInitialCycle(ctx, q, cfg, st, scenarios); err != nil {
			return Summary{}, fmt.Errorf("pipeline: initial cycle for case %s: %w", st.caseID, err)
		}
	}

	// Cycle 2: MRD tier 3's literal "nudge ignored once -> incentivized_nudge"
	// trigger, over the cases that actually reached that state in cycle 1.
	for _, st := range states {
		if st.lastTier == strategy.TierNudge && st.lastOutcome == execution.OutcomeNoResponse {
			if err := runUpgradeCycle(ctx, q, cfg, st, scenarios); err != nil {
				return Summary{}, fmt.Errorf("pipeline: upgrade cycle for case %s: %w", st.caseID, err)
			}
		}
	}

	return summarize(states), nil
}

func runInitialCycle(ctx context.Context, q Queries, cfg Config, st *caseState, scenarios map[string]string) error {
	diagResult, err := diagnosis.Classify(st.event, st.caseID)
	if err != nil {
		// A data/config gap (unmapped failure_code), not an infra failure —
		// fail this one case safely rather than aborting the whole batch.
		if logErr := audit.Log(ctx, q, audit.LogEntry{
			CaseID: st.caseID, AgentName: audit.AgentDiagnosis, Decision: "error",
			Alternatives: []string{err.Error()},
		}); logErr != nil {
			return logErr
		}
		return q.UpdateCase(ctx, st.caseID, "", 0, "", "error")
	}
	st.diagResult = diagResult
	confidence := diagResult.Confidence
	if err := audit.Log(ctx, q, audit.LogEntry{
		CaseID: st.caseID, AgentName: audit.AgentDiagnosis, Decision: diagResult.RootCause,
		Confidence: &confidence, Alternatives: diagResult.Evidence,
	}); err != nil {
		return err
	}

	decision := strategy.SelectTier(diagResult, st.event.HistoryScore)
	if err := logStrategy(ctx, q, decision); err != nil {
		return err
	}

	return processDecision(ctx, q, cfg, st, decision, scenarios)
}

func runUpgradeCycle(ctx context.Context, q Queries, cfg Config, st *caseState, scenarios map[string]string) error {
	upgraded := strategy.NudgeIgnoredUpgrade(st.caseID)
	if err := logStrategy(ctx, q, upgraded); err != nil {
		return err
	}
	return processDecision(ctx, q, cfg, st, upgraded, scenarios)
}

func logStrategy(ctx context.Context, q Queries, decision strategy.StrategyDecision) error {
	return audit.Log(ctx, q, audit.LogEntry{
		CaseID: decision.CaseID, AgentName: audit.AgentStrategy, Decision: decision.Tier,
		Alternatives: decision.Alternatives,
	})
}

// processDecision runs the shared guardrail -> execution -> promise ->
// audit -> case-update tail, used by both the initial and the upgrade
// cycle. Guardrail always runs immediately before Execution here — there is
// no code path to Execution that skips it.
func processDecision(ctx context.Context, q Queries, cfg Config, st *caseState, decision strategy.StrategyDecision, scenarios map[string]string) error {
	gResult := guardrail.Check(decision, st.event, cfg.Caps, cfg.SimulatedHour, st.attempts)
	if err := audit.Log(ctx, q, audit.LogEntry{
		CaseID: st.caseID, AgentName: audit.AgentGuardrail, Decision: guardrailDecisionLabel(gResult),
		Alternatives: gResult.Evidence,
	}); err != nil {
		return err
	}

	outcome, err := execution.Simulate(gResult, decision, st.event, scenarios)
	if err != nil {
		// execution.ErrGuardrailNotVerified (or any other Simulate error)
		// fails this one case safely rather than crashing the batch run.
		outcome = execution.ExecutionOutcome{CaseID: st.caseID, Result: "internal_error", Evidence: []string{err.Error()}}
	}
	if err := audit.Log(ctx, q, audit.LogEntry{
		CaseID: st.caseID, AgentName: audit.AgentExecution, Decision: outcome.Result,
		Alternatives: outcome.Evidence,
	}); err != nil {
		return err
	}

	if decision.Channel != strategy.ChannelNone && gResult.Allowed && !gResult.Held {
		st.attempts++
	}

	promiseResult, err := promise.RecordIfAny(ctx, q, outcome, st.event.TransactionID, st.caseID, cfg.SimulatedToday, scenarios)
	if err != nil {
		return fmt.Errorf("promise tracking: %w", err)
	}
	if err := audit.Log(ctx, q, audit.LogEntry{
		CaseID: st.caseID, AgentName: audit.AgentPromise,
		Decision:     fmt.Sprintf("strikeCount=%d escalated=%t", promiseResult.StrikeCount, promiseResult.Escalated),
		Alternatives: promiseResult.Evidence,
	}); err != nil {
		return err
	}

	// promise.RecordIfAny may have just escalated this case (2 broken-promise
	// strikes) directly in the DB via EscalateCase. Reflect that here too —
	// otherwise this UpdateCase call would run after EscalateCase's and
	// silently overwrite tier_chosen back to the pre-escalation tier while
	// status correctly stayed 'escalated', leaving the row internally
	// inconsistent.
	finalTier := gResult.FinalTier
	if promiseResult.Escalated {
		finalTier = strategy.TierEscalate
	}

	status := statusFor(outcome, promiseResult)
	if err := q.UpdateCase(ctx, st.caseID, st.diagResult.RootCause, st.diagResult.Confidence, finalTier, status); err != nil {
		return err
	}

	st.lastTier = finalTier
	st.lastOutcome = outcome.Result
	st.lastAmount = outcome.Amount
	return nil
}

func guardrailDecisionLabel(g guardrail.GuardrailResult) string {
	switch {
	case g.Held:
		return fmt.Sprintf("held rule=%s", g.RuleFired)
	case !g.Allowed:
		return fmt.Sprintf("blocked rule=%s tier=%s", g.RuleFired, g.FinalTier)
	default:
		return fmt.Sprintf("allowed tier=%s", g.FinalTier)
	}
}

func statusFor(outcome execution.ExecutionOutcome, promiseResult promise.PromiseResult) string {
	if promiseResult.Escalated || outcome.Result == execution.OutcomeEscalated {
		return "escalated"
	}
	switch outcome.Result {
	case execution.OutcomeRecovered:
		return "recovered"
	case execution.OutcomeHeld:
		return "held"
	default:
		return "open"
	}
}

func summarize(states []*caseState) Summary {
	summary := Summary{TotalCases: len(states), TierBreakdown: map[string]int{}}
	for _, st := range states {
		summary.TierBreakdown[st.lastTier]++
		summary.MoneyRecovered += st.lastAmount
	}
	return summary
}
