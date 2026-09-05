package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vasooli/internal/guardrail"
	"vasooli/internal/pipeline"
)

// /api/simulate needs no DB at all — Server.Queries is left nil here on
// purpose, and the test still runs unconditionally (no DATABASE_URL skip),
// proving the endpoint really is stateless rather than just documented as
// such.
func newSimulateOnlyServer() *Server {
	return NewServer(nil, pipeline.Config{Caps: guardrail.DefaultCaps})
}

func postSimulate(t *testing.T, mux *http.ServeMux, body SimulateRequestDTO) (*httptest.ResponseRecorder, SimulateResponseDTO) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshaling request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/simulate", bytes.NewReader(raw))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var resp SimulateResponseDTO
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decoding response %q: %v", rec.Body.String(), err)
		}
	}
	return rec, resp
}

func TestSimulate_DisputedFlagOverridesEverything(t *testing.T) {
	server := newSimulateOnlyServer()
	mux := http.NewServeMux()
	server.Routes(mux)

	// A disputed flag on an otherwise-perfectly-fixable case (card_expired,
	// high history score) must still lock to escalate — the exact scenario
	// a judge would try first to "break" the simulator.
	rec, resp := postSimulate(t, mux, SimulateRequestDTO{
		FailureCode: "CARD_EXPIRED", HistoryScore: 90, DisputedFlag: true, SimulatedHour: 12,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if resp.RootCause != "disputed" {
		t.Errorf("expected root cause disputed, got %s", resp.RootCause)
	}
	if resp.GuardrailAllowed {
		t.Error("expected guardrail to block a disputed case")
	}
	if resp.GuardrailFinalTier != "escalate" {
		t.Errorf("expected final tier escalate, got %s", resp.GuardrailFinalTier)
	}
	if resp.GuardrailRuleFired != "dispute_flag" {
		t.Errorf("expected rule dispute_flag, got %s", resp.GuardrailRuleFired)
	}
}

func TestSimulate_NormalCaseAllowed(t *testing.T) {
	server := newSimulateOnlyServer()
	mux := http.NewServeMux()
	server.Routes(mux)

	rec, resp := postSimulate(t, mux, SimulateRequestDTO{
		FailureCode: "CARD_EXPIRED", HistoryScore: 60, SimulatedHour: 12,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if resp.TierChosen != "nudge" || resp.GuardrailFinalTier != "nudge" {
		t.Errorf("expected nudge tier throughout, got chosen=%s final=%s", resp.TierChosen, resp.GuardrailFinalTier)
	}
	if !resp.GuardrailAllowed {
		t.Errorf("expected allowed, got blocked/held: rule=%s reason=%s", resp.GuardrailRuleFired, resp.GuardrailReason)
	}
	if len(resp.StrategyAlternatives) != 3 {
		t.Errorf("expected 3 rejected-alternative reasons, got %d", len(resp.StrategyAlternatives))
	}
}

func TestSimulate_OutsideContactHoursIsHeldNotEscalated(t *testing.T) {
	server := newSimulateOnlyServer()
	mux := http.NewServeMux()
	server.Routes(mux)

	rec, resp := postSimulate(t, mux, SimulateRequestDTO{
		FailureCode: "CARD_EXPIRED", HistoryScore: 60, SimulatedHour: 3, // 3am, outside 9-20 window
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if resp.GuardrailAllowed {
		t.Error("expected outside-hours contact to be blocked")
	}
	if !resp.GuardrailHeld {
		t.Error("expected Held=true, not a hard escalate, for the contact-hours rule")
	}
	if resp.GuardrailFinalTier != "nudge" {
		t.Errorf("held must not escalate the tier, got final tier %s", resp.GuardrailFinalTier)
	}
}

func TestSimulate_UnmappedFailureCodeReturns400NotFabricated(t *testing.T) {
	server := newSimulateOnlyServer()
	mux := http.NewServeMux()
	server.Routes(mux)

	rec, _ := postSimulate(t, mux, SimulateRequestDTO{
		FailureCode: "NOT_A_REAL_CODE", HistoryScore: 60, SimulatedHour: 12,
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unmapped failure code, got %d: %s", rec.Code, rec.Body.String())
	}

	var errBody ErrorDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decoding error response: %v", err)
	}
	if errBody.Error == "" {
		t.Error("expected a non-empty error message")
	}
}

func TestSimulate_MalformedJSONReturns400(t *testing.T) {
	server := newSimulateOnlyServer()
	mux := http.NewServeMux()
	server.Routes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/simulate", bytes.NewReader([]byte("{not json")))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d", rec.Code)
	}
}
