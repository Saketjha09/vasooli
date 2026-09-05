package detector

import (
	"context"
	"testing"
	"time"

	"vasooli/internal/db"
)

type fakeQueries struct {
	rows []db.TransactionWithCustomer
}

func (f fakeQueries) ListTransactionsWithCustomer(ctx context.Context) ([]db.TransactionWithCustomer, error) {
	return f.rows, nil
}

func TestIngestBatch_LoadsFullBatchWithNoFiltering(t *testing.T) {
	now := time.Now()
	fake := fakeQueries{rows: []db.TransactionWithCustomer{
		{TransactionID: "t1", CustomerID: "c1", Amount: 100, FailureCode: "GATEWAY_TIMEOUT", CreatedAt: now, HistoryScore: 70, DisputedFlag: false, DoNotContact: false},
		{TransactionID: "t2", CustomerID: "c2", Amount: 200, FailureCode: "CARD_EXPIRED", CreatedAt: now, HistoryScore: 60, DisputedFlag: true, DoNotContact: false},
		{TransactionID: "t3", CustomerID: "c3", Amount: 300, FailureCode: "INSUFFICIENT_FUNDS", CreatedAt: now, HistoryScore: 50, DisputedFlag: false, DoNotContact: true},
	}}

	events, err := IngestBatch(context.Background(), fake)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != len(fake.rows) {
		t.Fatalf("expected all %d rows to pass through untouched, got %d", len(fake.rows), len(events))
	}

	for i, e := range events {
		want := fake.rows[i]
		if e.TransactionID != want.TransactionID ||
			e.CustomerID != want.CustomerID ||
			e.Amount != want.Amount ||
			e.FailureCode != want.FailureCode ||
			e.HistoryScore != want.HistoryScore ||
			e.DisputedFlag != want.DisputedFlag ||
			e.DoNotContact != want.DoNotContact {
			t.Fatalf("event %d does not match source row: got %+v, want fields from %+v", i, e, want)
		}
	}
}

func TestIngestBatch_EmptyBatch(t *testing.T) {
	events, err := IngestBatch(context.Background(), fakeQueries{rows: nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events for empty batch, got %d", len(events))
	}
}
