package db

import "context"

// CaseRow is one case as the API layer needs it: the case's own fields plus
// its recovered amount, derived from that case's most recent execution
// audit_log entry (0/NULL if it never reached a "recovered" outcome).
type CaseRow struct {
	CaseID        string
	TransactionID string
	RootCause     string
	Confidence    float64
	TierChosen    string
	Status        string
	Amount        float64
}

// ApiQueries is the subset of DB operations the API layer depends on, kept
// as its own interface (same pattern as every other stage's *Queries type).
type ApiQueries interface {
	SummarizeCases(ctx context.Context) (totalCases int, tierBreakdown map[string]int, moneyRecovered float64, err error)
	ListCasesSummary(ctx context.Context) ([]CaseRow, error)
	GetCaseByID(ctx context.Context, caseID string) (CaseRow, error)
	GetDisputedCaseID(ctx context.Context) (string, error)
}

// caseRecoveredAmountJoin is shared by every query below that needs a
// case's recovered amount: each case's most recent execution audit_log
// row, defaulting to 0 when that row's decision wasn't "recovered" (or the
// case never reached execution at all).
const caseRecoveredAmountJoin = `
	LEFT JOIN LATERAL (
		SELECT amount, decision
		FROM audit_log a
		WHERE a.case_id = c.id AND a.agent_name = 'execution'
		ORDER BY a.timestamp DESC, a.id DESC
		LIMIT 1
	) latest_execution ON true
`

func (d *DB) SummarizeCases(ctx context.Context) (int, map[string]int, float64, error) {
	var totalCases int
	if err := d.Pool.QueryRow(ctx, "SELECT count(*) FROM cases").Scan(&totalCases); err != nil {
		return 0, nil, 0, err
	}

	tierBreakdown := make(map[string]int)
	rows, err := d.Pool.Query(ctx, `
		SELECT COALESCE(tier_chosen, ''), count(*) FROM cases GROUP BY tier_chosen
	`)
	if err != nil {
		return 0, nil, 0, err
	}
	for rows.Next() {
		var tier string
		var count int
		if err := rows.Scan(&tier, &count); err != nil {
			rows.Close()
			return 0, nil, 0, err
		}
		tierBreakdown[tier] = count
	}
	if err := rows.Err(); err != nil {
		return 0, nil, 0, err
	}

	var moneyRecovered float64
	err = d.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM (
			SELECT DISTINCT ON (case_id) case_id, decision, amount
			FROM audit_log
			WHERE agent_name = 'execution'
			ORDER BY case_id, timestamp DESC, id DESC
		) latest
		WHERE decision = 'recovered'
	`).Scan(&moneyRecovered)
	if err != nil {
		return 0, nil, 0, err
	}

	return totalCases, tierBreakdown, moneyRecovered, nil
}

func (d *DB) ListCasesSummary(ctx context.Context) ([]CaseRow, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT c.id, c.transaction_id, COALESCE(c.root_cause, ''), COALESCE(c.confidence, 0),
		       COALESCE(c.tier_chosen, ''), c.status, COALESCE(latest_execution.amount, 0)
		FROM cases c
		`+caseRecoveredAmountJoin+`
		ORDER BY c.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CaseRow
	for rows.Next() {
		var r CaseRow
		if err := rows.Scan(&r.CaseID, &r.TransactionID, &r.RootCause, &r.Confidence, &r.TierChosen, &r.Status, &r.Amount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (d *DB) GetCaseByID(ctx context.Context, caseID string) (CaseRow, error) {
	var r CaseRow
	err := d.Pool.QueryRow(ctx, `
		SELECT c.id, c.transaction_id, COALESCE(c.root_cause, ''), COALESCE(c.confidence, 0),
		       COALESCE(c.tier_chosen, ''), c.status, COALESCE(latest_execution.amount, 0)
		FROM cases c
		`+caseRecoveredAmountJoin+`
		WHERE c.id = $1
	`, caseID).Scan(&r.CaseID, &r.TransactionID, &r.RootCause, &r.Confidence, &r.TierChosen, &r.Status, &r.Amount)
	return r, err
}

// GetDisputedCaseID returns the case ID for the guardrail spotlight: the
// case whose transaction's customer has disputed_flag=true. Returns
// pgx.ErrNoRows (via the underlying Scan) if none exists.
func (d *DB) GetDisputedCaseID(ctx context.Context) (string, error) {
	var caseID string
	err := d.Pool.QueryRow(ctx, `
		SELECT c.id
		FROM cases c
		JOIN transactions t ON t.id = c.transaction_id
		JOIN customers cu ON cu.id = t.customer_id
		WHERE cu.disputed_flag = true
		ORDER BY c.id
		LIMIT 1
	`).Scan(&caseID)
	return caseID, err
}
