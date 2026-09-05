// Package detector implements the Detector Agent (MRD 4.1, PRD FR1): it loads
// the full synthetic batch from Postgres with no manual intervention and
// shapes each row into the Event the rest of the pipeline consumes.
package detector

import (
	"context"
	"time"

	"vasooli/internal/db"
)

// Event is one transaction plus the customer context every downstream agent
// needs, so Diagnosis/Strategy/Guardrail never have to re-query the DB.
type Event struct {
	TransactionID string
	CustomerID    string
	Amount        float64
	FailureCode   string
	CreatedAt     time.Time
	HistoryScore  int
	DisputedFlag  bool
	DoNotContact  bool
}

// IngestBatch loads every transaction in the demo batch, in deterministic
// (created_at, id) order, with no filtering or classification — that belongs
// to the Diagnosis Agent.
func IngestBatch(ctx context.Context, q db.Queries) ([]Event, error) {
	rows, err := q.ListTransactionsWithCustomer(ctx)
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(rows))
	for _, r := range rows {
		events = append(events, Event{
			TransactionID: r.TransactionID,
			CustomerID:    r.CustomerID,
			Amount:        r.Amount,
			FailureCode:   r.FailureCode,
			CreatedAt:     r.CreatedAt,
			HistoryScore:  r.HistoryScore,
			DisputedFlag:  r.DisputedFlag,
			DoNotContact:  r.DoNotContact,
		})
	}
	return events, nil
}
