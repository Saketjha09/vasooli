# Vasooli — Product Requirements Document
### Razorpay AI Buildathon, Track 03: AI Revenue Recovery

*Companion to the MRD/architecture doc — this covers what to build, acceptance criteria, and delivery order.*

---

## 1. Product Goal

Ship a working system that ingests a batch of failed/at-risk synthetic transactions, autonomously diagnoses root cause, takes a graduated bounded recovery action per case, and presents a full reasoning trace + money-recovered summary in a live dashboard — within the hackathon window.

## 2. Users / Personas (for framing, not literal build targets)

- **Merchant ops person** — wants to see money recovered and exceptions without digging through logs
- **Compliance/finance reviewer** — needs every automated action to be explainable and bounded
- **Judge** — needs to understand, in under 3 minutes, what the system decided and why, including the one case it correctly refused to act on

## 3. Functional Requirements

### FR1 — Event Ingestion
- System loads a synthetic batch (30–50 transactions) covering all failure types listed in the MRD
- Each transaction has: amount, customer history score, failure code, dispute flag, do-not-contact flag, timestamp

### FR2 — Root Cause Diagnosis
- Every transaction is assigned exactly one root cause label + a confidence score (0–1)
- Diagnosis output must include the evidence/features used to reach that label (for audit trail)

### FR3 — Graduated Recovery Decision
- Every diagnosed case is assigned exactly one tier: silent retry / nudge / incentivized nudge / escalate
- Tier assignment logic must be inspectable — no undocumented LLM-only judgment calls with no rule backing

### FR4 — Policy Guardrails
- Every proposed action is checked against the fixed caps before execution:
  - max 3 contact attempts per case
  - max 10% discount value
  - dispute flag → hard block + auto-escalate
  - do-not-contact flag → hard block
  - outside allowed contact hours (simulated) → hold, don't cancel
- A blocked action must log which rule fired and why

### FR5 — Mocked Execution
- Chosen action is simulated (no real Razorpay calls) and produces a synthetic outcome: recovered / no response / broken promise
- Execution outcome updates the transaction status and feeds the money-recovered total

### FR6 — Promise-to-Pay Tracking
- Customer commitments are logged with a promised date
- A broken promise (missed date) increments a strike count; 2 strikes → auto-escalate

### FR7 — Audit Log
- Every agent decision across the pipeline is persisted with: input snapshot, decision, confidence, rejected alternatives, policy rule fired (if any), timestamp
- Audit log must be queryable per case (for the dashboard drill-down)

### FR8 — Dashboard
- **Summary view:** total cases, money recovered, tier breakdown chart
- **Case detail view:** step-by-step reasoning chain rendered visually (not a raw log dump)
- **Guardrail spotlight:** dedicated surfacing of the disputed-case lockout example

## 4. Non-Functional Requirements

- **Explainability:** no action in the demo should require narrating "trust me, the model decided this" — every decision must have a visible reason on screen
- **Determinism for demo safety:** the exact demo dataset must produce the same tier outcomes every run (avoid flaky LLM-only branching logic during judging)
- **Latency:** batch of ~40 cases should process and render in well under a minute — judges won't wait
- **No real payment/PII data:** all data synthetic, clearly labeled as such

## 5. Acceptance Criteria (per agent, MRD section 4)

| Agent | Done when... |
|---|---|
| Detector | Loads full synthetic batch without manual intervention |
| Diagnosis | Every case has a root cause + confidence + evidence list |
| Strategy | Every case has exactly one tier, traceable to diagnosis output |
| Policy Guardrail | Disputed case is blocked and escalated; caps are enforced and logged |
| Execution | Every action produces a logged synthetic outcome |
| Promise Tracker | Broken-promise case reaches escalation after 2 strikes in the demo data |
| Audit Log | Any case's full decision chain is retrievable and displayed |
| Dashboard | Summary + case drill-down + guardrail spotlight all render from real pipeline output, not hardcoded mockups |

## 6. Build Milestones (so something demoable exists at every checkpoint)

1. **Milestone 1 — Data + pipeline skeleton:** synthetic dataset defined, Postgres schema live, Detector → Diagnosis → Strategy running end-to-end with placeholder logic (even if diagnosis is just rule-based lookup at first)
2. **Milestone 2 — Guardrails + execution:** Policy Guardrail Agent wired in, mocked Execution Agent producing outcomes, disputed-case lockout working
3. **Milestone 3 — Promise tracking + audit log:** commitments logged, broken-promise escalation working, full audit trail persisted per case
4. **Milestone 4 — Dashboard:** summary view, case detail reasoning trace, guardrail spotlight
5. **Milestone 5 — Polish for demo:** confirm deterministic outcomes on the exact demo dataset, rehearse the 5-step demo script from the MRD

If time runs short, Milestones 1–3 are the non-negotiable core (they carry the explainability/bounded/gated story). Milestone 4 can degrade to a simpler table view if needed — the dashboard is presentation, not the differentiator itself.

## 7. Out of Scope

- Real Razorpay API integration
- Voice/TTS recovery channel
- Tiered/dynamic negotiation limits (fixed caps only)
- Multi-merchant support
- Authentication/user management

## 8. Success Metrics (for your own tracking, not necessarily shown to judges)

- % of synthetic cases correctly routed to the "obviously correct" tier (sanity-check your own logic)
- Disputed case always blocked, 0% leakage
- Full reasoning chain renders for 100% of cases in the dashboard

## 9. Open Decisions Carried from MRD

- Diagnosis Agent: rule-based core + LLM-generated explanation text (recommended) vs LLM-driven classification
- Policy caps: hardcoded constants vs a small config table (recommended: config table)
