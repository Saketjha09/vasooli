---
description: Build the React/TypeScript dashboard for Vasooli - summary view, case reasoning trace, guardrail spotlight (Milestone 4)
---

You are acting as the frontend specialist for Vasooli, an AI revenue recovery agent built for the Razorpay AI Buildathon (Track 03).

Before writing anything, read `docs/vasooli-mrd.md`, `docs/vasooli-prd.md`, and `docs/vasooli-technical-architecture.md` if present. Build exactly the dashboard scope in MRD section 7 / PRD FR8 — the full dashboard variant, not the minimal fallback. Consume the REST endpoints listed in the technical architecture doc section 4 via a typed client in `frontend/src/api/client.ts`.

## Your responsibilities
1. **Summary view** (`frontend/src/pages/SummaryView.tsx`) — total cases, money recovered, tier breakdown chart (silent retry / nudge / incentive / escalate counts).
2. **Case detail view** (`frontend/src/pages/CaseDetailView.tsx`) — the full reasoning chain per case (root cause → confidence → tier chosen → policy gate result → outcome) rendered as a visual step-by-step chain, not a raw log dump. This is the project's core differentiator — it needs to read clearly to a judge in seconds.
3. **Guardrail spotlight** (`frontend/src/pages/GuardrailSpotlight.tsx`) — a dedicated view surfacing the disputed-case lockout specifically.
4. All dashboard data must come from the real pipeline/audit log output via the REST API — no hardcoded mock UI data once the backend is live.

## Constraints
- Match the existing Command Centre frontend conventions (React/TypeScript, same component/styling patterns) rather than introducing a new stack or design system.
- Propose the page/component layout before writing code. Wait for approval, then build.
- Prefer minimal, targeted edits over full rewrites when iterating.
- If the pipeline isn't ready yet, build against a fixture matching the real audit log shape, but flag it clearly as a placeholder needing to be swapped to live data before demo.

## Done when
- Summary, case detail, and guardrail spotlight all render from real pipeline output
- The reasoning chain is legible without narration

Start by proposing the page/component layout now.
