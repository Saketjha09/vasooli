# Vasooli — Technical Architecture
### Companion to the MRD (product/conceptual architecture) and PRD (requirements) — this covers implementation structure

---

## 1. Service Boundaries & Canonical File Structure

**One Go backend service**, not six microservices — cleaner deployment, faster to build in a hackathon window, and the internal separation is enough to keep the "explainable, bounded, gated" story intact.

This is the exact repo layout — every subagent and any Claude Code session should treat this as canonical, so nothing gets created in an inconsistent location:

```
vasooli/
├── .claude/
│   └── agents/
│       ├── data-schema-agent.md
│       ├── pipeline-logic-agent.md
│       ├── dashboard-agent.md
│       └── qa-demo-agent.md
│
├── docs/
│   ├── vasooli-mrd.md
│   ├── vasooli-prd.md
│   └── vasooli-technical-architecture.md
│
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── detector/
│   │   │   └── detector.go
│   │   ├── diagnosis/
│   │   │   └── diagnosis.go
│   │   ├── strategy/
│   │   │   └── strategy.go
│   │   ├── guardrail/
│   │   │   ├── guardrail.go
│   │   │   └── caps_config.go        # config table for the fixed caps
│   │   ├── execution/
│   │   │   └── execution.go
│   │   ├── promise/
│   │   │   └── promise.go
│   │   ├── audit/
│   │   │   └── audit.go
│   │   ├── pipeline/
│   │   │   └── pipeline.go           # orchestrator - calls stages in order
│   │   ├── api/
│   │   │   ├── handlers.go
│   │   │   └── dto.go                # request/response shapes
│   │   └── db/
│   │       ├── db.go
│   │       └── queries.go
│   ├── migrations/
│   │   ├── 0001_init_schema.sql
│   │   └── ...
│   ├── fixtures/
│   │   └── demo_dataset.json         # fixed synthetic batch, deterministic
│   ├── go.mod
│   └── go.sum
│
├── frontend/
│   ├── src/
│   │   ├── pages/
│   │   │   ├── SummaryView.tsx
│   │   │   ├── CaseDetailView.tsx
│   │   │   └── GuardrailSpotlight.tsx
│   │   ├── components/
│   │   │   ├── ReasoningChain.tsx
│   │   │   ├── TierBreakdownChart.tsx
│   │   │   └── ...
│   │   ├── api/
│   │   │   └── client.ts             # typed client for backend REST endpoints
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   └── vite.config.ts (or equivalent)
│
├── .env.example                       # DATABASE_URL, ALLOWED_ORIGIN, PORT, DEMO_SEED_PATH
├── .gitignore
└── README.md
```

**Fixed conventions for any Claude Code session working in this repo:**
- Docs live in `docs/` — always read `docs/vasooli-mrd.md`, `docs/vasooli-prd.md`, and this file before making architectural decisions.
- Subagent configs live in `.claude/agents/` at the repo root — this is a fixed Claude Code convention, not a project choice, and must not move.
- Backend code lives under `backend/`, frontend under `frontend/` — never mix Go and TS files across these boundaries.
- New files should match the tree above exactly; if a subagent needs a path not shown here, it should flag that and ask rather than inventing a new top-level folder.

## 2. Pipeline Orchestration

The `/internal/pipeline` package runs each case through the 6 stages **in-process, sequentially**, per case, when a batch is triggered:

```
for each transaction in batch:
    event      := detector.Ingest(transaction)
    diagnosis  := diagnosis.Classify(event)
    tier       := strategy.SelectTier(diagnosis, customerHistory)
    decision   := guardrail.Check(tier, customer, caps)   // may override tier to "blocked/escalate"
    outcome    := execution.Simulate(decision)
    promise.RecordIfAny(outcome)
    audit.Log(event, diagnosis, tier, decision, outcome)
```

No case moves to `execution` without passing through `guardrail` first — this ordering is the literal implementation of "bounded and gated" and should never be reordered.

## 3. API Contracts Between Stages (internal Go structs, not HTTP)

```go
type DiagnosisResult struct {
    CaseID     string
    RootCause  string   // transient_gateway | card_expired | insufficient_funds |
                         // checkout_friction | willful_nonpayment | disputed
    Confidence float64
    Evidence   []string
}

type StrategyDecision struct {
    CaseID string
    Tier   string // silent_retry | nudge | incentivized_nudge | escalate
    Channel string // sms | email | in_app
}

type GuardrailResult struct {
    CaseID      string
    Allowed     bool
    FinalTier   string // may differ from StrategyDecision.Tier if blocked/escalated
    RuleFired   string // empty if not blocked
    Reason      string
}

type ExecutionOutcome struct {
    CaseID  string
    Result  string // recovered | no_response | broken_promise
    Amount  float64
}
```

## 4. REST API (backend → dashboard)

| Endpoint | Method | Purpose |
|---|---|---|
| `/api/batch/run` | POST | Trigger a pipeline run over the fixed demo dataset |
| `/api/batch/summary` | GET | Total cases, money recovered, tier breakdown |
| `/api/cases` | GET | List of cases with high-level status |
| `/api/cases/:id` | GET | Full reasoning chain for one case (diagnosis → tier → guardrail → outcome) |
| `/api/cases/:id/audit` | GET | Raw audit log entries for one case |
| `/api/guardrail/spotlight` | GET | The disputed-case lockout example, pre-filtered for the dashboard callout |
| `/api/guardrail/policy` | GET | The current fixed caps (max contact attempts, max discount %, contact-hours window) for the dashboard's policy panel — sourced from `guardrail.Caps`, not a DB query |

All endpoints read from Postgres via `/internal/db` — no direct frontend-to-Supabase connection, per your call above. `/api/guardrail/policy` is the one exception: it's read-only config display, sourced directly from the running server's `guardrail.Caps`, not persisted state.

## 5. Deployment Topology

```
 [Vercel: React dashboard]
          │  HTTPS (REST calls)
          ▼
 [Railway: Go backend service]
          │  Postgres wire protocol
          ▼
 [Supabase: Postgres DB]
```

Same topology as Command Centre — reuse existing env var patterns for `DATABASE_URL`, CORS origins, etc. Since CORS/IPv6 issues came up before on that project, set the allowed frontend origin explicitly in the Go server config early rather than debugging it under demo pressure.

## 6. Environment Variables (backend)

- `DATABASE_URL` — Supabase Postgres connection string
- `ALLOWED_ORIGIN` — Vercel frontend URL, for CORS
- `PORT` — server port (Railway sets this automatically)
- `DEMO_SEED_PATH` — path to the fixed synthetic dataset fixture file

## 7. Ownership Mapping (which subagent touches what)

| Path | Owner |
|---|---|
| `/migrations`, `/fixtures` | `data-schema-agent` |
| `/internal/detector` through `/internal/audit`, `/internal/pipeline`, `/internal/api` | `pipeline-logic-agent` |
| `/frontend/*` | `dashboard-agent` |
| none (read-only verification) | `qa-demo-agent` |

Keeping this mapping explicit avoids two subagents editing the same files in parallel.
