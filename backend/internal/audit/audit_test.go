package audit

import (
	"context"
	"testing"

	"vasooli/internal/db"
)

type fakeAuditQueries struct {
	entries []db.AuditEntry
	nextID  int
}

func (f *fakeAuditQueries) InsertAuditEntry(ctx context.Context, caseID, agentName, decision string, confidence *float64, alternatives []string, amount *float64) error {
	f.nextID++
	f.entries = append(f.entries, db.AuditEntry{
		ID:               string(rune('a' + f.nextID)),
		CaseID:           caseID,
		AgentName:        agentName,
		Decision:         decision,
		Confidence:       confidence,
		AlternativesJSON: alternatives,
		Amount:           amount,
	})
	return nil
}

func (f *fakeAuditQueries) ListAuditLogForCase(ctx context.Context, caseID string) ([]db.AuditEntry, error) {
	var out []db.AuditEntry
	for _, e := range f.entries {
		if e.CaseID == caseID {
			out = append(out, e)
		}
	}
	return out, nil
}

func TestLog_PersistsEntry(t *testing.T) {
	fake := &fakeAuditQueries{}
	confidence := 0.95

	err := Log(context.Background(), fake, LogEntry{
		CaseID:       "case-1",
		AgentName:    AgentDiagnosis,
		Decision:     "card_expired",
		Confidence:   &confidence,
		Alternatives: []string{"failure_code=CARD_EXPIRED"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fake.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(fake.entries))
	}
	if fake.entries[0].Decision != "card_expired" {
		t.Errorf("decision not persisted correctly: %+v", fake.entries[0])
	}
}

func TestListForCase_RetrievesFullChainForOneCaseOnly(t *testing.T) {
	fake := &fakeAuditQueries{}
	ctx := context.Background()

	entries := []LogEntry{
		{CaseID: "case-A", AgentName: AgentDiagnosis, Decision: "card_expired"},
		{CaseID: "case-A", AgentName: AgentStrategy, Decision: "nudge", Alternatives: []string{"silent_retry: not applicable"}},
		{CaseID: "case-A", AgentName: AgentGuardrail, Decision: "allowed"},
		{CaseID: "case-A", AgentName: AgentExecution, Decision: "recovered"},
		{CaseID: "case-B", AgentName: AgentDiagnosis, Decision: "disputed"}, // different case, must not leak in
	}
	for _, e := range entries {
		if err := Log(ctx, fake, e); err != nil {
			t.Fatalf("unexpected error logging %+v: %v", e, err)
		}
	}

	chain, err := ListForCase(ctx, fake, "case-A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chain) != 4 {
		t.Fatalf("expected 4 entries for case-A, got %d", len(chain))
	}
	wantAgents := []string{AgentDiagnosis, AgentStrategy, AgentGuardrail, AgentExecution}
	for i, e := range chain {
		if e.AgentName != wantAgents[i] {
			t.Errorf("entry %d: expected agent %s, got %s", i, wantAgents[i], e.AgentName)
		}
		if e.CaseID != "case-A" {
			t.Errorf("entry %d: case-B data leaked into case-A's chain: %+v", i, e)
		}
	}
}
