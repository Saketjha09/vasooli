package db

import (
	"context"
	"encoding/json"
	"time"
)

// TransactionWithCustomer is the joined row shape Detector needs — one row
// per transaction, with the customer fields Diagnosis/Strategy/Guardrail
// need later folded in so no agent has to join separately.
type TransactionWithCustomer struct {
	TransactionID string
	CustomerID    string
	Amount        float64
	FailureCode   string
	CreatedAt     time.Time
	HistoryScore  int
	DisputedFlag  bool
	DoNotContact  bool
}

// Queries is the subset of DB operations agents depend on, kept as an
// interface so pipeline stages can be tested against a fake.
type Queries interface {
	ListTransactionsWithCustomer(ctx context.Context) ([]TransactionWithCustomer, error)
}

func (d *DB) ListTransactionsWithCustomer(ctx context.Context) ([]TransactionWithCustomer, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT t.id, t.customer_id, t.amount, t.failure_code, t.created_at,
		       c.history_score, c.disputed_flag, c.do_not_contact
		FROM transactions t
		JOIN customers c ON c.id = t.customer_id
		ORDER BY t.created_at, t.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TransactionWithCustomer
	for rows.Next() {
		var r TransactionWithCustomer
		if err := rows.Scan(
			&r.TransactionID, &r.CustomerID, &r.Amount, &r.FailureCode, &r.CreatedAt,
			&r.HistoryScore, &r.DisputedFlag, &r.DoNotContact,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// PromiseQueries is the subset of DB operations the Promise-to-Pay Tracker
// depends on, kept as its own interface (same pattern as Queries above) so
// it can be faked in tests without a live DB.
type PromiseQueries interface {
	CountBrokenPromises(ctx context.Context, caseID string) (int, error)
	InsertPromise(ctx context.Context, caseID string, promisedDate time.Time, kept bool) error
	EscalateCase(ctx context.Context, caseID string) error
}

func (d *DB) CountBrokenPromises(ctx context.Context, caseID string) (int, error) {
	var count int
	err := d.Pool.QueryRow(ctx, `
		SELECT count(*) FROM promises WHERE case_id = $1 AND kept = false
	`, caseID).Scan(&count)
	return count, err
}

func (d *DB) InsertPromise(ctx context.Context, caseID string, promisedDate time.Time, kept bool) error {
	_, err := d.Pool.Exec(ctx, `
		INSERT INTO promises (case_id, promised_date, kept) VALUES ($1, $2, $3)
	`, caseID, promisedDate, kept)
	return err
}

func (d *DB) EscalateCase(ctx context.Context, caseID string) error {
	_, err := d.Pool.Exec(ctx, `
		UPDATE cases SET tier_chosen = 'escalate', status = 'escalated' WHERE id = $1
	`, caseID)
	return err
}

// AuditEntry is one row of the audit_log table, per the MRD 4.7 shape:
// input snapshot is implicit in Decision/Confidence/Alternatives being the
// exact values the agent computed, not a separate re-serialized blob.
type AuditEntry struct {
	ID               string
	CaseID           string
	AgentName        string
	Decision         string
	Confidence       *float64
	AlternativesJSON []string
	Timestamp        time.Time
}

// AuditQueries is the subset of DB operations the Audit Log agent depends
// on, kept as its own interface (same pattern as Queries/PromiseQueries).
type AuditQueries interface {
	InsertAuditEntry(ctx context.Context, caseID, agentName, decision string, confidence *float64, alternatives []string) error
	ListAuditLogForCase(ctx context.Context, caseID string) ([]AuditEntry, error)
}

func (d *DB) InsertAuditEntry(ctx context.Context, caseID, agentName, decision string, confidence *float64, alternatives []string) error {
	altJSON, err := json.Marshal(alternatives)
	if err != nil {
		return err
	}
	_, err = d.Pool.Exec(ctx, `
		INSERT INTO audit_log (case_id, agent_name, decision, confidence, alternatives_json)
		VALUES ($1, $2, $3, $4, $5)
	`, caseID, agentName, decision, confidence, altJSON)
	return err
}

func (d *DB) ListAuditLogForCase(ctx context.Context, caseID string) ([]AuditEntry, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT id, case_id, agent_name, decision, confidence, alternatives_json, timestamp
		FROM audit_log
		WHERE case_id = $1
		ORDER BY timestamp, id
	`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var altJSON []byte
		if err := rows.Scan(&e.ID, &e.CaseID, &e.AgentName, &e.Decision, &e.Confidence, &altJSON, &e.Timestamp); err != nil {
			return nil, err
		}
		if len(altJSON) > 0 {
			if err := json.Unmarshal(altJSON, &e.AlternativesJSON); err != nil {
				return nil, err
			}
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// CaseQueries is the subset of DB operations the pipeline orchestrator
// depends on for case lifecycle management (creation, updates after each
// cycle, and the full-reset idempotency strategy), kept as its own
// interface (same pattern as Queries/PromiseQueries/AuditQueries).
type CaseQueries interface {
	CreateCase(ctx context.Context, transactionID string) (string, error)
	UpdateCase(ctx context.Context, caseID, rootCause string, confidence float64, tier, status string) error
	ResetDerivedState(ctx context.Context) error
}

func (d *DB) CreateCase(ctx context.Context, transactionID string) (string, error) {
	var id string
	err := d.Pool.QueryRow(ctx, `
		INSERT INTO cases (transaction_id, status) VALUES ($1, 'processing') RETURNING id
	`, transactionID).Scan(&id)
	return id, err
}

func (d *DB) UpdateCase(ctx context.Context, caseID, rootCause string, confidence float64, tier, status string) error {
	_, err := d.Pool.Exec(ctx, `
		UPDATE cases SET root_cause = $2, confidence = $3, tier_chosen = $4, status = $5 WHERE id = $1
	`, caseID, rootCause, confidence, tier, status)
	return err
}

// ResetDerivedState clears every pipeline-output table (audit_log,
// policy_events, promises, cases — child-before-parent for FK safety) while
// leaving the seed data (customers, transactions) untouched. Run at the top
// of every batch run so repeated demo runs produce identical results rather
// than accumulating duplicate cases/strikes across runs.
//
// Deliberately 4 separate statements, not one multi-statement string: pgx's
// pooled connections use the extended query protocol, which does not
// support multiple commands in a single Exec call.
func (d *DB) ResetDerivedState(ctx context.Context) error {
	stmts := []string{
		"DELETE FROM audit_log",
		"DELETE FROM policy_events",
		"DELETE FROM promises",
		"DELETE FROM cases",
	}
	for _, stmt := range stmts {
		if _, err := d.Pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
