# Vasooli — AI Revenue Recovery Agent
### MRD + Architecture — Razorpay AI Buildathon, Track 03

---

## 1. Problem Statement

Revenue leaks out of a merchant's business through failed payments, checkout abandonment, failed subscription renewals, and overdue B2B receivables. Most recovery tooling today treats every failure the same way: retry once, send a generic email, and move on. This ignores that different failures have different *causes*, and different causes need different *interventions*.

**Track's own framing (why this matters to judges):** revenue loss is a chain — payment degrades → root cause → recovery action → money recovered. Most hackathon entries will skip the middle two steps and build a shallow retry bot.

## 2. Core Differentiator

**Vasooli doesn't just retry — it diagnoses, then makes a graduated, bounded decision, and shows its reasoning for every case.**

Instead of binary retry/no-retry, each failed transaction moves through a visible causal chain:

```
Symptom → Root Cause (with confidence) → Chosen Action → Policy Gate → Outcome
```

And the chosen action is one of four graduated tiers, not a single hammer:

| Tier | Trigger | Action |
|---|---|---|
| 1. Silent retry | Transient cause (gateway timeout, bank server issue) | Auto-retry, no customer contact |
| 2. Nudge | Fixable cause (card expired, insufficient funds) | Send update-payment-method message |
| 3. Incentivized nudge | Nudge ignored once | Offer bounded discount / grace period (hard-capped) |
| 4. Escalate to human | Repeated broken promise, dispute flag, or policy cap hit | Route to human agent, freeze automated contact |

This directly maps to the track's stated judging bar: **"every money action explainable, bounded and gated. Show the audit trail and one failure handled gracefully."**

## 3. The "Graceful Failure" Demo Moment

One synthetic case in the batch will have an active dispute/chargeback flag. The Policy Guardrail Agent detects this *before* any recovery action fires, locks the case, logs the reason, and routes it to escalation — instead of contacting a customer who has a live dispute (which would be compliance-unsafe). This is the single moment judges will remember: the system refusing to act, correctly.

## 4. Agent Architecture

```
                    ┌─────────────────┐
   synthetic        │  Detector Agent │
   event feed  ───▶ │  (ingest events)│
                    └────────┬────────┘
                             ▼
                    ┌─────────────────┐
                    │ Diagnosis Agent │  → root cause + confidence score
                    └────────┬────────┘
                             ▼
                    ┌─────────────────┐
                    │ Strategy Agent  │  → picks tier 1-4 action
                    └────────┬────────┘
                             ▼
                    ┌───────────────────────┐
                    │ Policy Guardrail Agent│  → hard caps, dispute lock,
                    │                       │     no-contact list, quiet hrs
                    └────────┬──────────────┘
                             ▼
                    ┌─────────────────┐
                    │ Execution Agent │  → mocked action (simulated call)
                    └────────┬────────┘
                             ▼
                    ┌────────────────────────┐
                    │ Promise-to-Pay Tracker │  → logs commitments, schedules
                    └────────┬───────────────┘   follow-up
                             ▼
                    ┌────────────────────────┐
                    │ Audit / Reasoning Log  │  → every decision, confidence,
                    └────────┬───────────────┘   rejected alternatives
                             ▼
                    ┌─────────────────┐
                    │  Dashboard (UI) │  → reasoning trace + money recovered
                    └─────────────────┘
```

### 4.1 Detector Agent
Ingests a synthetic event stream: payment failures, checkout abandonments, subscription dunning events, overdue invoices. Each event carries mocked metadata (amount, customer history, failure code, timestamp).

### 4.2 Diagnosis Agent
Classifies root cause from failure code + context: `transient_gateway`, `card_expired`, `insufficient_funds`, `checkout_friction`, `willful_nonpayment`, `disputed`. Outputs a confidence score (0–1) and the evidence used.

### 4.3 Strategy Agent
Maps `(root cause, confidence, customer history)` → one of the 4 tiers. Also decides channel (SMS / email / in-app — kept text-based since voice is out of scope).

### 4.4 Policy Guardrail Agent — the "bounded and gated" core
Fixed hard caps (configurable but static for MVP):
- Max 3 automated contact attempts per case
- Max discount offer: 10% of transaction value
- No contact if `disputed = true` → auto-lock + escalate
- No contact outside 9am–8pm (simulated clock)
- No contact if customer is on a do-not-contact list

Every action from the Strategy Agent passes through this agent before execution. If blocked, the block reason is logged and the case is escalated.

### 4.5 Execution Agent
Simulates the action (mocked Razorpay call, mocked message send) and logs a synthetic outcome (success / no response / broken promise).

### 4.6 Promise-to-Pay Tracker
For any case where a customer commits to a payment date, logs the promise and schedules a follow-up check. If broken twice, escalates automatically.

### 4.7 Audit / Reasoning Log
The backbone of the explainability story. Every agent's decision is stored with: input, decision, confidence, alternatives considered and rejected, policy rule that gated it (if any), timestamp.

## 5. Data Model (Postgres — matches your existing stack)

- `transactions` — id, merchant_id, customer_id, amount, status, failure_code, created_at
- `customers` — id, name, history_score, disputed_flag, do_not_contact
- `cases` — id, transaction_id, root_cause, confidence, tier_chosen, status
- `policy_events` — id, case_id, rule_fired, action_blocked (bool), reason
- `promises` — id, case_id, promised_date, kept (bool/null)
- `audit_log` — id, case_id, agent_name, decision, confidence, alternatives_json, timestamp

## 6. Tech Stack (reusing what you already know)

- **Backend:** Go — each agent as a discrete service/function, orchestrated by a simple pipeline (no need for a heavy agent framework given mocked data)
- **DB:** PostgreSQL (Supabase, as with Command Centre)
- **Frontend:** React/TypeScript — dashboard with reasoning trace UI
- **LLM calls (if any):** used only inside Diagnosis/Strategy agents for reasoning generation; keep deterministic fallback logic so the demo never breaks on an API hiccup
- **Deployment:** Railway (backend) + Vercel (frontend), matching Command Centre setup

## 7. Dashboard Scope (full, per your call)

- **Batch summary view:** total cases processed, money recovered, tier breakdown (how many silent retries / nudges / incentives / escalations)
- **Case detail view:** the full reasoning trace per case — root cause, confidence, tier chosen, policy gate result, outcome — rendered as a visual step-by-step chain (this is your key differentiator, make it visually clear, not a log dump)
- **Guardrail highlight:** a dedicated view or callout for the disputed-case lockout, so judges see the "graceful failure" moment without digging

## 8. Synthetic Dataset (for demo)

~30–50 mocked transactions covering all root causes, including:
- A few resolvable by silent retry
- A few resolvable by nudge
- A few needing incentive
- At least one repeated-broken-promise case → escalation
- **Exactly one disputed-flag case** → the guardrail lockout showcase

## 9. MVP Cut Lines

**In scope:**
- All 6 agents, pipeline-orchestrated
- Fixed-cap policy guardrails
- Full dashboard with reasoning trace
- Synthetic data, mocked Razorpay calls

**Out of scope (for hackathon window):**
- Real Razorpay API integration
- Voice/TTS channel
- Tiered/dynamic negotiation limits
- Multi-merchant support (single merchant context is fine)

## 10. Demo Script (for judging)

1. Show batch of 40 synthetic cases loading in
2. Show dashboard summary: money recovered, tier breakdown
3. Drill into 2–3 individual cases showing the reasoning chain
4. Drill into the disputed-case lockout — narrate why the agent refused to act
5. Show the audit log for any one case — full transparency, no black box

## 11. Open Decisions Still Needed
- Exact LLM vs rule-based split for Diagnosis Agent reasoning generation
- Whether policy caps are hardcoded constants or a small config table (recommend config table — looks more "product-grade" to judges for near-zero extra effort)
