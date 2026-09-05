// Package api implements the REST layer (architecture doc section 4): thin
// HTTP handlers translating pipeline/DB output into JSON. No business logic
// lives here — every decision was already made by the 6 pipeline agents;
// this package only shapes and serves what they already persisted.
package api

import "time"

// BatchRunResponseDTO is the response to POST /api/batch/run — the rollup
// pipeline.RunBatch returns directly, without a separate DB query, since
// RunBatch just computed it from the run it performed.
type BatchRunResponseDTO struct {
	// TotalCases is the number of transactions processed in this run.
	TotalCases int `json:"totalCases"`
	// TierBreakdown counts cases by their FINAL tier after all cycles
	// (e.g. a case upgraded nudge -> incentivized_nudge is counted once,
	// under "incentivized_nudge"). Keys are the strategy.Tier* constants:
	// "silent_retry", "nudge", "incentivized_nudge", "escalate".
	TierBreakdown map[string]int `json:"tierBreakdown"`
	// MoneyRecovered is the sum of recovered amounts across all cases in
	// this run (post-discount where a discount applied).
	MoneyRecovered float64 `json:"moneyRecovered"`
}

// BatchSummaryDTO is the response to GET /api/batch/summary — the same
// shape as BatchRunResponseDTO, but always freshly queried from the DB
// (cases + audit_log) rather than held over from the last RunBatch call, so
// it stays correct even after a server restart.
type BatchSummaryDTO struct {
	TotalCases     int            `json:"totalCases"`
	TierBreakdown  map[string]int `json:"tierBreakdown"`
	MoneyRecovered float64        `json:"moneyRecovered"`
}

// CaseSummaryDTO is one row of GET /api/cases — the list view. Amount is
// the recovered amount if the case's most recent execution outcome was
// "recovered" (0 otherwise: no money has moved for that case, whether
// because it's still open, held, escalated, or a promise was broken).
// TransactionAmount is the original transaction amount regardless of
// outcome — needed for "amount at risk" style aggregates the dashboard
// can't compute from Amount alone.
type CaseSummaryDTO struct {
	CaseID            string  `json:"caseId"`
	TransactionID     string  `json:"transactionId"`
	RootCause         string  `json:"rootCause"`
	Confidence        float64 `json:"confidence"`
	TierChosen        string  `json:"tierChosen"`
	Status            string  `json:"status"` // "processing" | "recovered" | "held" | "open" | "escalated" | "error"
	Amount            float64 `json:"amount"`
	TransactionAmount float64 `json:"transactionAmount"`
}

// ReasoningStepDTO is ONE entry from that case's audit_log, in the exact
// order the agent ran. This is a flat, chronological list — not a fixed
// one-step-per-agent structure. Most cases have exactly 6 steps (one per
// agent: diagnosis, strategy, guardrail, execution, promise — detector
// itself isn't case-scoped, so it never appears here). A case that goes
// through MRD tier 3's "nudge ignored once -> incentivized_nudge" upgrade
// (architecture doc's second pipeline cycle) has a SECOND full pass
// appended after the first: another strategy, guardrail, execution, and
// promise step, in the same flat list — i.e. two entries with
// AgentName="strategy" at different positions, not a nested "cycles"
// structure. Render this as a simple ordered timeline; do not assume any
// fixed count or one-entry-per-agent-name.
type ReasoningStepDTO struct {
	AgentName string `json:"agentName"` // "diagnosis" | "strategy" | "guardrail" | "execution" | "promise"
	Decision  string `json:"decision"`
	// Confidence is present only for diagnosis (a genuine 0-1 confidence
	// score); null for every other agent, which are rule/policy checks
	// with no scored confidence — not zero, actually absent.
	Confidence *float64 `json:"confidence,omitempty"`
	// Alternatives holds different content per agent (see the audit
	// package's doc comment for the full rationale):
	//   - strategy: a genuine one-line "why not" per tier NOT chosen
	//     (e.g. "silent_retry: not applicable — root cause is
	//     card_expired, not transient").
	//   - diagnosis/guardrail/execution/promise: the evidence this agent's
	//     decision was actually based on, including checks that passed
	//     (e.g. guardrail's "discountPct=8.0, cap=10.0 (within cap)"),
	//     not a fabricated list of rejected candidates — these agents are
	//     rule lookups, not multi-candidate scored decisions.
	Alternatives []string `json:"alternatives"`
	// Amount is present only on an execution step, and only when its
	// Decision is "recovered" — null otherwise.
	Amount    *float64  `json:"amount,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// CaseDetailDTO is the response to both GET /api/cases/:id and GET
// /api/guardrail/spotlight (the latter is this same shape, pre-filtered to
// the one disputed-flag case). ReasoningChain is the full, ordered
// diagnosis -> ... -> promise decision trail — see ReasoningStepDTO's doc
// comment for how a cycle-2 (tier-3 upgrade) case is represented in it.
// TransactionAmount is the original transaction amount regardless of
// outcome — Amount alone is the recovered amount (0 for unresolved cases),
// the wrong number for anything that needs the case's real dollar value.
type CaseDetailDTO struct {
	CaseID            string             `json:"caseId"`
	TransactionID     string             `json:"transactionId"`
	RootCause         string             `json:"rootCause"`
	Confidence        float64            `json:"confidence"`
	TierChosen        string             `json:"tierChosen"`
	Status            string             `json:"status"`
	Amount            float64            `json:"amount"`
	TransactionAmount float64            `json:"transactionAmount"`
	ReasoningChain    []ReasoningStepDTO `json:"reasoningChain"`
}

// GuardrailPolicyDTO is the response to GET /api/guardrail/policy — the
// fixed caps guardrail.Check enforces on every case, exposed read-only for
// the dashboard's policy panel. Mirrors guardrail.Caps field-for-field.
type GuardrailPolicyDTO struct {
	MaxContactAttempts int     `json:"maxContactAttempts"`
	MaxDiscountPct     float64 `json:"maxDiscountPct"`
	ContactWindowStart int     `json:"contactWindowStart"` // hour, 0-23, inclusive
	ContactWindowEnd   int     `json:"contactWindowEnd"`   // hour, 0-23, exclusive
}

// AuditEntryDTO is one row of GET /api/cases/:id/audit — the raw audit log,
// minimally shaped (an ID and CaseID added, since this endpoint is meant as
// the un-opinionated full record, unlike ReasoningStepDTO which is already
// framed for the dashboard's reasoning-chain UI).
type AuditEntryDTO struct {
	ID           string    `json:"id"`
	CaseID       string    `json:"caseId"`
	AgentName    string    `json:"agentName"`
	Decision     string    `json:"decision"`
	Confidence   *float64  `json:"confidence,omitempty"`
	Alternatives []string  `json:"alternatives"`
	Amount       *float64  `json:"amount,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// SimulateRequestDTO is the body of POST /api/simulate — a hypothetical
// case's inputs, not a real transaction. FailureCode is required unless
// DisputedFlag is true (diagnosis.Classify checks disputed status first,
// unconditionally, before ever looking at the failure code).
type SimulateRequestDTO struct {
	FailureCode          string `json:"failureCode"`
	HistoryScore         int    `json:"historyScore"`
	DisputedFlag         bool   `json:"disputedFlag"`
	DoNotContact         bool   `json:"doNotContact"`
	SimulatedHour        int    `json:"simulatedHour"`
	PriorContactAttempts int    `json:"priorContactAttempts"`
}

// SimulateResponseDTO is the response to POST /api/simulate: the genuine
// output of diagnosis.Classify -> strategy.SelectTier -> guardrail.Check
// run against the hypothetical inputs, nothing persisted. TierChosen is
// Strategy's own pick before guardrail; GuardrailFinalTier may differ if
// guardrail blocked or held it.
type SimulateResponseDTO struct {
	RootCause            string   `json:"rootCause"`
	Confidence           float64  `json:"confidence"`
	DiagnosisEvidence    []string `json:"diagnosisEvidence"`
	TierChosen           string   `json:"tierChosen"`
	StrategyAlternatives []string `json:"strategyAlternatives"`
	GuardrailAllowed     bool     `json:"guardrailAllowed"`
	GuardrailHeld        bool     `json:"guardrailHeld"`
	GuardrailFinalTier   string   `json:"guardrailFinalTier"`
	GuardrailRuleFired   string   `json:"guardrailRuleFired"`
	GuardrailReason      string   `json:"guardrailReason"`
	GuardrailEvidence    []string `json:"guardrailEvidence"`
}

// ErrorDTO is the JSON body for any non-2xx response.
type ErrorDTO struct {
	Error string `json:"error"`
}
