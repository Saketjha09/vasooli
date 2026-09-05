# Vasooli

AI revenue recovery agent built for the **Razorpay AI Buildathon — Track 03: AI Revenue Recovery**.

Instead of a single retry-or-email bot, Vasooli diagnoses *why* a payment is at risk, then makes a graduated, bounded, and fully explainable recovery decision — silent retry, nudge, incentivized nudge, or escalate — with every step of its reasoning logged and auditable.

Full context lives in [`docs/`](./docs):
- [`vasooli-mrd.md`](./docs/vasooli-mrd.md) — problem, differentiator, agent architecture
- [`vasooli-prd.md`](./docs/vasooli-prd.md) — requirements, acceptance criteria, build milestones
- [`vasooli-technical-architecture.md`](./docs/vasooli-technical-architecture.md) — service boundaries, API contracts, canonical file tree

## Stack

- **Backend:** Go, Postgres (via Supabase)
- **Frontend:** React + TypeScript
- **Deployment:** Railway (backend) + Vercel (frontend) + Supabase (Postgres)

## Repo structure

See the canonical file tree in `docs/vasooli-technical-architecture.md` (section 1) for the full layout. At a glance:

```
backend/    → Go service: 6-agent pipeline (Detector, Diagnosis, Strategy,
              Guardrail, Execution, Promise Tracker) + Audit Log + REST API
frontend/   → React dashboard: batch summary, per-case reasoning trace,
              guardrail spotlight
docs/       → MRD, PRD, technical architecture
.claude/    → Claude Code agent configs (agents/) and slash commands (commands/)
```

## Setup

1. Copy `.env.example` to `.env` and fill in real values. **Never commit `.env`.**
2. Backend:
   ```
   cd backend
   go mod tidy
   go run cmd/server/main.go
   ```
3. Frontend:
   ```
   cd frontend
   npm install
   npm run dev
   ```

## Build status

Development follows the milestone order in `docs/vasooli-prd.md` section 6, built component-by-component via Claude Code (see `.claude/commands/`):

- [x] Milestone 1 — Postgres schema + synthetic dataset
- [x] Milestone 1–3 — Pipeline agents (Detector, Diagnosis, Strategy, Guardrail, Execution, Promise Tracker, Audit Log)
- [ ] Pipeline orchestrator wiring all 6 agents + audit together per case
- [ ] Milestone 4 — Dashboard
- [ ] Milestone 5 — QA against PRD acceptance criteria + demo rehearsal

## Development workflow

This project is built with Claude Code using project-scoped slash commands in `.claude/commands/` (e.g. `/data-schema-agent`, `/pipeline-logic-agent`, `/dashboard-agent`, `/qa-demo-agent`), each scoped to a specific part of the codebase and required to propose an approach before writing code. See `.claude/commands/README.md` for details.
