package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vasooli/internal/db"
	"vasooli/internal/guardrail"
	"vasooli/internal/pipeline"
)

// setupTestDB connects to a real Postgres (DATABASE_URL), resets it to a
// clean schema, and applies every migration in backend/migrations — same
// pattern used to hand-verify every earlier component, but committed here
// as an automated test per the approved httptest-against-live-Postgres
// approach for this component. Skips (not fails) if DATABASE_URL isn't set,
// so `go test ./...` still passes with no Postgres available.
func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping live-Postgres API test")
	}

	ctx := context.Background()
	conn, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	t.Cleanup(conn.Close)

	if _, err := conn.Pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
		t.Fatalf("resetting schema: %v", err)
	}

	migrationsDir := "../../migrations"
	files := []string{"0001_init_schema.sql", "0002_seed_demo_data.sql", "0003_add_audit_log_amount.sql"}
	for _, f := range files {
		raw, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			t.Fatalf("reading migration %s: %v", f, err)
		}
		// pgx's pooled connections use the extended query protocol, which
		// (like ResetDerivedState in internal/db) doesn't support multiple
		// commands in one Exec call — strip full-line "--" comments, then
		// split into individual statements. A naive split on ";" is safe
		// for these specific migration files (no semicolons appear inside
		// string literals).
		var noComments strings.Builder
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "--") {
				continue
			}
			noComments.WriteString(line)
			noComments.WriteByte('\n')
		}
		for _, stmt := range strings.Split(noComments.String(), ";") {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := conn.Pool.Exec(ctx, stmt); err != nil {
				t.Fatalf("applying migration %s, statement %q: %v", f, stmt, err)
			}
		}
	}

	return conn
}

func newTestServer(conn *db.DB) *Server {
	cfg := pipeline.Config{
		Caps:           guardrail.DefaultCaps,
		SimulatedHour:  12,
		SimulatedToday: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
		ScenariosPath:  "../../fixtures/demo_dataset.json",
	}
	return NewServer(conn, cfg)
}

func doRequest(t *testing.T, mux *http.ServeMux, method, path string, target any) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if target != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
			t.Fatalf("%s %s: decoding response %q: %v", method, path, rec.Body.String(), err)
		}
	}
	return rec
}

func TestAPI_FullBatchLifecycle(t *testing.T) {
	conn := setupTestDB(t)
	server := newTestServer(conn)
	mux := http.NewServeMux()
	server.Routes(mux)

	var runResp BatchRunResponseDTO
	rec := doRequest(t, mux, "POST", "/api/batch/run", &runResp)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/batch/run: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if runResp.TotalCases != 40 {
		t.Errorf("expected 40 total cases, got %d", runResp.TotalCases)
	}
	if runResp.MoneyRecovered <= 0 {
		t.Errorf("expected some money recovered, got %v", runResp.MoneyRecovered)
	}

	var summary BatchSummaryDTO
	rec = doRequest(t, mux, "GET", "/api/batch/summary", &summary)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/batch/summary: expected 200, got %d", rec.Code)
	}
	if summary.TotalCases != runResp.TotalCases || summary.MoneyRecovered != runResp.MoneyRecovered {
		t.Errorf("GET /api/batch/summary disagrees with the run response: %+v vs %+v", summary, runResp)
	}

	var cases []CaseSummaryDTO
	rec = doRequest(t, mux, "GET", "/api/cases", &cases)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/cases: expected 200, got %d", rec.Code)
	}
	if len(cases) != 40 {
		t.Fatalf("expected 40 cases listed, got %d", len(cases))
	}
	for _, c := range cases {
		if c.CaseID == "" || c.TransactionID == "" || c.RootCause == "" {
			t.Errorf("case summary missing required fields: %+v", c)
		}
	}

	// Guardrail spotlight: the flagship "graceful failure" moment.
	var spotlight CaseDetailDTO
	rec = doRequest(t, mux, "GET", "/api/guardrail/spotlight", &spotlight)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/guardrail/spotlight: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if spotlight.TierChosen != "escalate" || spotlight.Status != "escalated" {
		t.Errorf("expected the disputed case to be escalate/escalated, got tier=%s status=%s", spotlight.TierChosen, spotlight.Status)
	}
	for _, step := range spotlight.ReasoningChain {
		if step.AgentName == "execution" {
			switch step.Decision {
			case "recovered", "no_response", "broken_promise":
				t.Errorf("disputed case must never reach a contact-tier execution outcome, got %s", step.Decision)
			}
		}
	}
	foundDisputeRule := false
	for _, step := range spotlight.ReasoningChain {
		if step.AgentName == "guardrail" {
			for _, alt := range step.Alternatives {
				if alt == "customer.disputed_flag=true" {
					foundDisputeRule = true
				}
			}
		}
	}
	if !foundDisputeRule {
		t.Error("expected the guardrail step's evidence to show customer.disputed_flag=true")
	}

	// Case detail + raw audit log agree, and the cycle-aware chain (2
	// strategy entries for a tier-3 upgraded case) is retrievable end to end.
	var detail CaseDetailDTO
	rec = doRequest(t, mux, "GET", "/api/cases/"+spotlight.CaseID, &detail)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/cases/:id: expected 200, got %d", rec.Code)
	}
	if detail.CaseID != spotlight.CaseID || len(detail.ReasoningChain) != len(spotlight.ReasoningChain) {
		t.Errorf("case detail for the same case ID should match the spotlight response")
	}

	var auditEntries []AuditEntryDTO
	rec = doRequest(t, mux, "GET", "/api/cases/"+spotlight.CaseID+"/audit", &auditEntries)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/cases/:id/audit: expected 200, got %d", rec.Code)
	}
	if len(auditEntries) != len(detail.ReasoningChain) {
		t.Errorf("raw audit log length (%d) should match reasoning chain length (%d)", len(auditEntries), len(detail.ReasoningChain))
	}

	// Not-found case: both /api/cases/:id and /api/cases/:id/audit must
	// agree on a clean 404, not leak the underlying Postgres SQLSTATE
	// 22P02 (invalid_text_representation) error a malformed UUID
	// path segment triggers.
	var caseErr ErrorDTO
	rec = doRequest(t, mux, "GET", "/api/cases/does-not-exist", &caseErr)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /api/cases/does-not-exist: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if caseErr.Error != "case not found" {
		t.Errorf("GET /api/cases/does-not-exist: expected clean 'case not found' message, got %q", caseErr.Error)
	}

	var auditErr ErrorDTO
	rec = doRequest(t, mux, "GET", "/api/cases/does-not-exist/audit", &auditErr)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /api/cases/does-not-exist/audit: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if auditErr.Error != "case not found" {
		t.Errorf("GET /api/cases/does-not-exist/audit: expected clean 'case not found' message, got %q (must not leak SQLSTATE)", auditErr.Error)
	}
}

func TestAPI_Idempotency(t *testing.T) {
	conn := setupTestDB(t)
	server := newTestServer(conn)
	mux := http.NewServeMux()
	server.Routes(mux)

	var first, second BatchRunResponseDTO
	doRequest(t, mux, "POST", "/api/batch/run", &first)
	doRequest(t, mux, "POST", "/api/batch/run", &second)

	if first.TotalCases != second.TotalCases || first.MoneyRecovered != second.MoneyRecovered {
		t.Fatalf("expected identical results across two /api/batch/run calls, got %+v vs %+v", first, second)
	}
}
