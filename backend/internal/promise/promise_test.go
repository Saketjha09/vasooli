package promise

import (
	"context"
	"testing"
	"time"

	"vasooli/internal/execution"
)

type fakePromiseQueries struct {
	brokenCount   int
	insertedDates []time.Time
	insertedKept  []bool
	escalated     bool
	escalatedFor  string
}

func (f *fakePromiseQueries) CountBrokenPromises(ctx context.Context, caseID string) (int, error) {
	return f.brokenCount, nil
}

func (f *fakePromiseQueries) InsertPromise(ctx context.Context, caseID string, promisedDate time.Time, kept bool) error {
	f.insertedDates = append(f.insertedDates, promisedDate)
	f.insertedKept = append(f.insertedKept, kept)
	f.brokenCount++ // keep the fake's count in sync with what a real COUNT(*) would return
	return nil
}

func (f *fakePromiseQueries) EscalateCase(ctx context.Context, caseID string) error {
	f.escalated = true
	f.escalatedFor = caseID
	return nil
}

var today = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

func TestRecordIfAny_NonBrokenPromiseOutcome_NoOp(t *testing.T) {
	fake := &fakePromiseQueries{}
	outcome := execution.ExecutionOutcome{CaseID: "c1", Result: execution.OutcomeRecovered}

	got, err := RecordIfAny(context.Background(), fake, outcome, "t1", "c1", today, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.PromiseCreated || got.Escalated || got.SeededPriorStrike {
		t.Fatalf("expected no-op for non-broken_promise outcome, got %+v", got)
	}
	if len(fake.insertedDates) != 0 {
		t.Fatalf("expected no DB writes, got %d inserts", len(fake.insertedDates))
	}
}

func TestRecordIfAny_UnscriptedBrokenPromise_OneStrikeNoEscalation(t *testing.T) {
	fake := &fakePromiseQueries{brokenCount: 0}
	outcome := execution.ExecutionOutcome{CaseID: "c2", Result: execution.OutcomeBrokenPromise}

	got, err := RecordIfAny(context.Background(), fake, outcome, "t2", "c2", today, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.PromiseCreated {
		t.Fatal("expected a promise to be recorded")
	}
	if got.SeededPriorStrike {
		t.Fatal("unscripted case must not get a seeded prior strike")
	}
	if got.StrikeCount != 1 {
		t.Fatalf("expected strike count 1, got %d", got.StrikeCount)
	}
	if got.Escalated {
		t.Fatal("1 strike must not escalate")
	}
}

func TestRecordIfAny_ScriptedRepeatedBrokenPromise_SeedsAndEscalatesAtTwoStrikes(t *testing.T) {
	fake := &fakePromiseQueries{brokenCount: 0}
	outcome := execution.ExecutionOutcome{CaseID: "c3", Result: execution.OutcomeBrokenPromise}
	scenarios := map[string]string{"t3": execution.ScenarioRepeatedBrokenPromise}

	got, err := RecordIfAny(context.Background(), fake, outcome, "t3", "c3", today, scenarios)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.SeededPriorStrike {
		t.Fatal("expected the scripted case to get a seeded prior strike")
	}
	if !got.PromiseCreated {
		t.Fatal("expected this cycle's live broken promise to also be recorded")
	}
	if got.StrikeCount != 2 {
		t.Fatalf("expected strike count 2 (1 seeded + 1 live), got %d", got.StrikeCount)
	}
	if !got.Escalated {
		t.Fatal("expected auto-escalation at 2 strikes")
	}
	if !fake.escalated || fake.escalatedFor != "c3" {
		t.Fatal("expected EscalateCase to be called for case c3")
	}

	if len(fake.insertedDates) != 2 {
		t.Fatalf("expected 2 promise rows inserted, got %d", len(fake.insertedDates))
	}
	if !fake.insertedDates[0].Before(fake.insertedDates[1]) {
		t.Fatalf("expected seeded promise dated before the live one: %v vs %v", fake.insertedDates[0], fake.insertedDates[1])
	}
	for _, k := range fake.insertedKept {
		if k {
			t.Fatal("both inserted promises must be kept=false (broken)")
		}
	}
}

func TestRecordIfAny_ScriptedCase_DoesNotReSeedOnSecondCall(t *testing.T) {
	// Simulates the case already having its seed (and presumably a prior
	// live strike) from an earlier call/run — seeding must be idempotent.
	fake := &fakePromiseQueries{brokenCount: 1}
	outcome := execution.ExecutionOutcome{CaseID: "c4", Result: execution.OutcomeBrokenPromise}
	scenarios := map[string]string{"t4": execution.ScenarioRepeatedBrokenPromise}

	got, err := RecordIfAny(context.Background(), fake, outcome, "t4", "c4", today, scenarios)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SeededPriorStrike {
		t.Fatal("must not re-seed when a prior broken promise already exists")
	}
	if len(fake.insertedDates) != 1 {
		t.Fatalf("expected exactly 1 insert (the live one only), got %d", len(fake.insertedDates))
	}
	if got.StrikeCount != 2 || !got.Escalated {
		t.Fatalf("expected 2 strikes and escalation, got count=%d escalated=%v", got.StrikeCount, got.Escalated)
	}
}
