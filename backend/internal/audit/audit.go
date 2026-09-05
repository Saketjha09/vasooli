// Package audit implements the Audit / Reasoning Log (MRD 4.7, PRD FR7): a
// thin persistence wrapper, not a recomputation layer. Every prior agent
// (Diagnosis, Strategy, Guardrail, Execution, Promise) already returns a
// decision + confidence (where applicable) + a reasoning trail; Log just
// writes one audit_log row per agent-per-case from those existing outputs.
//
// "Rejected alternatives" (FR7) is populated differently per agent, by
// design, not by omission:
//   - Strategy: a genuine one-line "why not" per non-chosen tier (its own
//     Alternatives field, derived from its rule table).
//   - Diagnosis, Guardrail, Execution, Promise: these are rule lookups /
//     policy checks, not multi-candidate scored decisions, so there are no
//     real "alternatives" to reject. Their own Evidence trail (what was
//     checked, including checks that passed) is passed through instead —
//     inventing a rejected-candidates list for them would be the
//     fabrication this design deliberately avoids.
package audit

import (
	"context"

	"vasooli/internal/db"
)

const (
	AgentDetector  = "detector"
	AgentDiagnosis = "diagnosis"
	AgentStrategy  = "strategy"
	AgentGuardrail = "guardrail"
	AgentExecution = "execution"
	AgentPromise   = "promise"
)

// LogEntry is what a caller (the pipeline orchestrator) hands to Log — the
// exact fields already computed by whichever agent just ran.
type LogEntry struct {
	CaseID       string
	AgentName    string
	Decision     string
	Confidence   *float64 // nil where an agent has no confidence score (Guardrail, Execution, Promise)
	Alternatives []string // see package doc for what this holds per agent
}

// Log persists one agent's decision for one case.
func Log(ctx context.Context, q db.AuditQueries, entry LogEntry) error {
	return q.InsertAuditEntry(ctx, entry.CaseID, entry.AgentName, entry.Decision, entry.Confidence, entry.Alternatives)
}

// ListForCase retrieves a case's full decision chain, in the order agents
// ran, per FR7's "queryable per case" requirement.
func ListForCase(ctx context.Context, q db.AuditQueries, caseID string) ([]db.AuditEntry, error) {
	return q.ListAuditLogForCase(ctx, caseID)
}
