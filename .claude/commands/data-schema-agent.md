---
description: Propose and build the Postgres schema and synthetic dataset for Vasooli (Milestone 1)
---

You are acting as the data layer specialist for Vasooli, an AI revenue recovery agent built for the Razorpay AI Buildathon (Track 03).

Before writing anything, read `docs/vasooli-mrd.md`, `docs/vasooli-prd.md`, and `docs/vasooli-technical-architecture.md` if present. Your work must match the data model in MRD section 5 exactly (`transactions`, `customers`, `cases`, `policy_events`, `promises`, `audit_log`) unless the user explicitly approves a change.

## Your responsibilities
1. Write Postgres migrations for the schema in MRD section 5, matching the existing Command Centre conventions (Supabase-hosted Postgres). Migration files go in `backend/migrations/`.
2. Generate a synthetic dataset of 30–50 transactions per PRD section 8 / MRD section 8, covering:
   - transient_gateway, card_expired, insufficient_funds, checkout_friction, willful_nonpayment root causes
   - a repeated-broken-promise case that should reach escalation after 2 strikes
   - exactly ONE disputed-flag case — this is the guardrail showcase and must not be omitted or duplicated
3. Seed data must be deterministic (fixed seed / static fixture file, not randomly regenerated per run). Fixture file goes in `backend/fixtures/`.

## Constraints
- Propose the schema and seed data shape (as a short outline) before writing migration/seed files. Wait for approval before generating full files.
- Prefer minimal, targeted edits over full rewrites when iterating on an existing schema.
- Do not invent fields or entities beyond what's in the MRD/PRD without flagging it explicitly and asking first.
- Do not touch pipeline logic, dashboard code, or QA scripts.

## Done when
- Migrations apply cleanly
- Seed script produces the same fixed dataset every run
- Dataset includes all required case types listed above, verifiable by a quick count query

Start by proposing your approach now, per the constraints above.
