package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vasooli/internal/audit"
	"vasooli/internal/db"
	"vasooli/internal/detector"
	"vasooli/internal/diagnosis"
	"vasooli/internal/guardrail"
	"vasooli/internal/pipeline"
	"vasooli/internal/strategy"
)

// invalidUUIDInputCode is Postgres SQLSTATE 22P02 (invalid_text_representation)
// — what cases.id = $1 raises when caseID isn't well-formed UUID text, e.g.
// a malformed or made-up path segment. Treated as "not found" (404) rather
// than an internal error (500): from an API consumer's perspective, a
// malformed ID and a nonexistent one both just mean "no such case."
const invalidUUIDInputCode = "22P02"

func isNotFound(err error) bool {
	if errors.Is(err, pgx.ErrNoRows) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == invalidUUIDInputCode
}

// Queries is everything the API layer needs from the DB: the pipeline's own
// Queries (RunBatch needs it) plus db.ApiQueries and db.AuditQueries for the
// read endpoints.
type Queries interface {
	pipeline.Queries
	db.ApiQueries
}

// Server holds the dependencies every handler needs.
type Server struct {
	Queries     Queries
	PipelineCfg pipeline.Config
}

func NewServer(q Queries, cfg pipeline.Config) *Server {
	return &Server{Queries: q, PipelineCfg: cfg}
}

// Routes registers every endpoint from architecture doc section 4 onto mux.
func (s *Server) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/batch/run", s.handleBatchRun)
	mux.HandleFunc("GET /api/batch/summary", s.handleBatchSummary)
	mux.HandleFunc("GET /api/cases", s.handleListCases)
	mux.HandleFunc("GET /api/cases/{id}", s.handleGetCase)
	mux.HandleFunc("GET /api/cases/{id}/audit", s.handleGetCaseAudit)
	mux.HandleFunc("GET /api/guardrail/spotlight", s.handleGuardrailSpotlight)
	mux.HandleFunc("GET /api/guardrail/policy", s.handleGuardrailPolicy)
	mux.HandleFunc("POST /api/simulate", s.handleSimulate)
}

func (s *Server) handleBatchRun(w http.ResponseWriter, r *http.Request) {
	summary, err := pipeline.RunBatch(r.Context(), s.Queries, s.PipelineCfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, BatchRunResponseDTO{
		TotalCases:     summary.TotalCases,
		TierBreakdown:  summary.TierBreakdown,
		MoneyRecovered: summary.MoneyRecovered,
	})
}

func (s *Server) handleBatchSummary(w http.ResponseWriter, r *http.Request) {
	totalCases, tierBreakdown, moneyRecovered, err := s.Queries.SummarizeCases(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, BatchSummaryDTO{
		TotalCases:     totalCases,
		TierBreakdown:  tierBreakdown,
		MoneyRecovered: moneyRecovered,
	})
}

func (s *Server) handleListCases(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Queries.ListCasesSummary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]CaseSummaryDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, caseRowToSummaryDTO(row))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetCase(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("id")
	dto, err := s.buildCaseDetail(r.Context(), caseID)
	if err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, errors.New("case not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

func (s *Server) handleGetCaseAudit(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("id")
	entries, err := audit.ListForCase(r.Context(), s.Queries, caseID)
	if err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, errors.New("case not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]AuditEntryDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, AuditEntryDTO{
			ID: e.ID, CaseID: e.CaseID, AgentName: e.AgentName, Decision: e.Decision,
			Confidence: e.Confidence, Alternatives: e.AlternativesJSON, Amount: e.Amount,
			Timestamp: e.Timestamp,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGuardrailSpotlight(w http.ResponseWriter, r *http.Request) {
	caseID, err := s.Queries.GetDisputedCaseID(r.Context())
	if err != nil {
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, errors.New("no disputed case found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	dto, err := s.buildCaseDetail(r.Context(), caseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

// handleGuardrailPolicy exposes the caps guardrail.Check actually enforces
// on every case — sourced from PipelineCfg.Caps (the same value RunBatch
// uses), not a separately hardcoded copy, so the dashboard can never drift
// from what's really enforced.
func (s *Server) handleGuardrailPolicy(w http.ResponseWriter, r *http.Request) {
	caps := s.PipelineCfg.Caps
	writeJSON(w, http.StatusOK, GuardrailPolicyDTO{
		MaxContactAttempts: caps.MaxContactAttempts,
		MaxDiscountPct:     caps.MaxDiscountPct,
		ContactWindowStart: caps.ContactWindowStart,
		ContactWindowEnd:   caps.ContactWindowEnd,
	})
}

// handleSimulate runs a hypothetical case's inputs through the real
// diagnosis.Classify -> strategy.SelectTier -> guardrail.Check chain —
// the exact same package functions the pipeline calls, not a second copy
// of the decision logic. Deliberately stateless: no case is created, no
// audit_log entry is written, nothing is persisted. Amount/TransactionID/
// CustomerID/CreatedAt are left zero-valued on the synthetic event since
// none of them affect diagnosis/strategy/guardrail's decisions — only
// Execution (never reached here) cares about amount.
func (s *Server) handleSimulate(w http.ResponseWriter, r *http.Request) {
	var req SimulateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid request body"))
		return
	}

	event := detector.Event{
		FailureCode:  req.FailureCode,
		HistoryScore: req.HistoryScore,
		DisputedFlag: req.DisputedFlag,
		DoNotContact: req.DoNotContact,
	}

	diag, err := diagnosis.Classify(event, "")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	decision := strategy.SelectTier(diag, req.HistoryScore)
	guard := guardrail.Check(decision, event, s.PipelineCfg.Caps, req.SimulatedHour, req.PriorContactAttempts)

	writeJSON(w, http.StatusOK, SimulateResponseDTO{
		RootCause: diag.RootCause, Confidence: diag.Confidence, DiagnosisEvidence: diag.Evidence,
		TierChosen: decision.Tier, StrategyAlternatives: decision.Alternatives,
		GuardrailAllowed: guard.Allowed, GuardrailHeld: guard.Held, GuardrailFinalTier: guard.FinalTier,
		GuardrailRuleFired: guard.RuleFired, GuardrailReason: guard.Reason, GuardrailEvidence: guard.Evidence,
	})
}

// buildCaseDetail is shared by GET /api/cases/:id and GET
// /api/guardrail/spotlight — the latter is just this same shape, pre-found
// by disputed_flag rather than by ID.
func (s *Server) buildCaseDetail(ctx context.Context, caseID string) (CaseDetailDTO, error) {
	row, err := s.Queries.GetCaseByID(ctx, caseID)
	if err != nil {
		return CaseDetailDTO{}, err
	}
	entries, err := audit.ListForCase(ctx, s.Queries, caseID)
	if err != nil {
		return CaseDetailDTO{}, err
	}
	chain := make([]ReasoningStepDTO, 0, len(entries))
	for _, e := range entries {
		chain = append(chain, ReasoningStepDTO{
			AgentName: e.AgentName, Decision: e.Decision, Confidence: e.Confidence,
			Alternatives: e.AlternativesJSON, Amount: e.Amount, Timestamp: e.Timestamp,
		})
	}
	dto := CaseDetailDTO{
		CaseID: row.CaseID, TransactionID: row.TransactionID, RootCause: row.RootCause,
		Confidence: row.Confidence, TierChosen: row.TierChosen, Status: row.Status,
		Amount: row.Amount, ReasoningChain: chain,
	}
	return dto, nil
}

func caseRowToSummaryDTO(row db.CaseRow) CaseSummaryDTO {
	return CaseSummaryDTO{
		CaseID: row.CaseID, TransactionID: row.TransactionID, RootCause: row.RootCause,
		Confidence: row.Confidence, TierChosen: row.TierChosen, Status: row.Status, Amount: row.Amount,
		TransactionAmount: row.TransactionAmount,
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, ErrorDTO{Error: err.Error()})
}

// CORSMiddleware sets Access-Control-Allow-Origin to allowedOrigin (the
// ALLOWED_ORIGIN env var per the architecture doc), explicitly rather than
// leaving CORS to be debugged under demo pressure (the same class of issue
// flagged as having come up before on Command Centre).
func CORSMiddleware(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
