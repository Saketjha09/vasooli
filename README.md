# Vasooli

AI revenue recovery agent built for the **Razorpay AI Buildathon — Track 03: AI Revenue Recovery**.

## Why "Vasooli"?

"Vasooli" (वसूली) is Hindi for "collection" or "recovery" — but colloquially, the word carries a harder edge: it's what you call rough, forceful, unbounded debt collection, the kind that doesn't ask permission and doesn't show its work. This project is deliberately the opposite of that. Every recovery action it takes is bounded by fixed policy caps, hard-locked the moment a dispute flag is raised, and logged step-by-step so nothing happens for a reason it can't show you. The name is the joke and the thesis in one word: this is *vasooli* with none of the roughness — collection that's compliant, explainable, and graceful even when it fails.

Instead of a single retry-or-email bot, Vasooli diagnoses *why* a payment is at risk, then makes a graduated, bounded, and fully explainable recovery decision — silent retry, nudge, incentivized nudge, or escalate — with every step of its reasoning logged and auditable. Beyond the core pipeline, the dashboard now includes a live policy-transparency panel, a case simulator for testing hypothetical scenarios against the real decision logic (with side-by-side comparison), and a per-case customer-message preview — so nothing about *what* the system would do, or *why*, is ever a black box.

Full context lives in [`docs/`](./docs):
- [`vasooli-mrd.md`](./docs/vasooli-mrd.md) — problem, differentiator, agent architecture
- [`vasooli-prd.md`](./docs/vasooli-prd.md) — requirements, acceptance criteria, build milestones
- [`vasooli-technical-architecture.md`](./docs/vasooli-technical-architecture.md) — service boundaries, API contracts, canonical file tree

## Stack

- **Backend:** Go, Postgres (via Supabase)
- **Frontend:** React + TypeScript + Tailwind (Inter typeface, centralized design tokens)
- **Deployment:** Railway (backend) + Vercel (frontend) + Supabase (Postgres)

## Repo structure

See the canonical file tree in `docs/vasooli-technical-architecture.md` (section 1) for the full layout. At a glance:

```
backend/    → Go service: 6-agent pipeline (Detector, Diagnosis, Strategy,
              Guardrail, Execution, Promise Tracker) + Audit Log + REST API
              (batch run/summary, case list/detail/audit, guardrail
              spotlight/policy, stateless case simulator)
frontend/   → React dashboard, sidebar-navigated multi-page app:
              layout/    → persistent Sidebar + Layout shell (every route)
              context/   → shared batch/case data (Overview + Cases pages)
              pages/     → OverviewPage, CasesPage, CaseDetailView,
                           GuardrailSpotlight, CaseSimulator
              components/→ ReasoningChain, CaseTable, charts, badges,
                           GuardrailPolicyCard, SimulatorPanel,
                           MessagePreview, and more
              lib/       → format/label helpers, message templates,
                           reasoning-chain mapping
              index.css  → centralized design tokens (palette, type scale,
                           radius) — the single source every component
                           styles against
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

Development followed the milestone order in `docs/vasooli-prd.md` section 6, built component-by-component via Claude Code (see `.claude/commands/`):

- [x] Milestone 1 — Postgres schema + synthetic dataset
- [x] Milestones 1–3 — Pipeline agents (Detector, Diagnosis, Strategy, Guardrail, Execution, Promise Tracker, Audit Log), orchestrator, and REST API
- [x] Milestone 4 — Dashboard
- [x] Milestone 5 — QA against PRD acceptance criteria + demo rehearsal

### Beyond the original MVP scope

The build grew past its original PRD definition once the core pipeline and dashboard were verified. See `docs/vasooli-prd.md` section 10 for the full detail; in short:

- **Guardrail Policy panel** — a live, always-current read-out of the fixed caps every decision is checked against, so "bounded" isn't just a claim in the docs.
- **Case Simulator (+ Compare mode)** — run the real diagnosis → strategy → guardrail chain against a hypothetical case, with a side-by-side two-scenario comparison, no data persisted. See `docs/simulator-field-reference.md` for what each field means and the exact thresholds/caps driving behavior.
- **Message Preview** — the actual templated customer-facing message a case's tier/channel would produce, rendered on Case Detail View.
- **Sidebar restructure** — the single-page summary view split into a persistent-sidebar, multi-page dashboard (Overview / Cases / Guardrail Spotlight / Simulator) with shared, freshness-aware data state.
- **Premium visual design system** — a centralized, validated color/typography token system (Inter, a 3-color status system, an ordinal severity ramp for tiers) applied consistently across every page.

## Development workflow

This project is built with Claude Code using project-scoped slash commands in `.claude/commands/` (e.g. `/data-schema-agent`, `/pipeline-logic-agent`, `/dashboard-agent`, `/qa-demo-agent`), each scoped to a specific part of the codebase and required to propose an approach before writing code. See `.claude/commands/README.md` for details. The original MVP build (Milestones 1–5) is complete; these commands remain the entry point for any further work in their respective territories.
