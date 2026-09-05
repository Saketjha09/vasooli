# Case Simulator — Field Reference

The Case Simulator (`/simulate`) runs a hypothetical case through the real
`diagnosis.Classify → strategy.SelectTier → guardrail.Check` pipeline — the
same code the batch pipeline uses, not a mock. Nothing is saved. This doc
explains what each input field does and the exact current thresholds/caps
that drive behavior, verified against the live code
(`backend/internal/diagnosis/diagnosis.go`, `backend/internal/strategy/strategy.go`,
`backend/internal/guardrail/guardrail.go`, `backend/internal/guardrail/caps_config.go`)
rather than assumed.

## Failure Code

The failure code is the raw signal the Diagnosis Agent maps to a root cause.
There are exactly 8 valid values:

| Failure Code | Root Cause | Confidence |
|---|---|---|
| `GATEWAY_TIMEOUT` | transient_gateway | 0.95 |
| `BANK_SERVER_ERROR` | transient_gateway | 0.95 |
| `CARD_EXPIRED` | card_expired | 0.95 |
| `INSUFFICIENT_FUNDS` | insufficient_funds | 0.95 |
| `CHECKOUT_ABANDONED` | checkout_friction | 0.90 |
| `OTP_TIMEOUT` | checkout_friction | 0.90 |
| `INVOICE_OVERDUE_NO_RESPONSE` | willful_nonpayment | 0.85 |
| `PAYMENT_DECLINED_REPEATED` | willful_nonpayment | 0.85 |

**Affected by Disputed?** All of them — if Disputed is checked, the failure
code is ignored entirely. `Classify` checks `event.DisputedFlag` first,
before the failure-code map is even consulted, and returns root cause
`disputed` unconditionally.

**Why this matters:** the root cause is what Strategy branches on — try the
same failure code with and without Disputed checked and watch the tier flip
from whatever that code normally produces straight to `escalate`, regardless
of which of the 8 codes you picked.

## History Score (0–100)

Represents the customer's payment reliability/history. It matters in exactly
one place in the whole pipeline: the `willful_nonpayment` branch of
`strategy.SelectTier`. Every other root cause ignores this field completely.

**Threshold (from `strategy.go`):** `willfulNonpaymentHistoryThreshold = 40`
- History Score `>= 40` → `incentivized_nudge` tier, email channel, 8% discount offer
- History Score `< 40` → `escalate`

**Why this matters:** set Failure Code to `INVOICE_OVERDUE_NO_RESPONSE` or
`PAYMENT_DECLINED_REPEATED`, then flip History Score from 39 to 40 — that
single point crosses the threshold and changes the outcome from escalation
to an automated incentive offer. For every other failure code, changing this
field does nothing observable.

## Disputed

A hard override, checked first, before anything else in the pipeline. When
true:
- Root cause is forced to `disputed` regardless of Failure Code (the
  dropdown is disabled in the UI to make this explicit).
- Guardrail's first rule (`dispute_flag`) fires immediately, before the
  do-not-contact check, the contact-attempts cap, the discount cap, or the
  contact-hours window are ever evaluated.
- Always routes to `escalate` with contact blocked.

**Why this matters:** this is the guardrail's zero-leakage showcase — no
combination of the other five fields can push a disputed case into any tier
other than a blocked escalation. Try maxing out every other field in the
case's favor (high History Score, in-window Hour, zero prior attempts) with
Disputed checked — the outcome doesn't move.

## Do Not Contact

A separate hard block, independent of Disputed. Guardrail's second rule
(`do_not_contact`) fires right after the dispute check, before the
contact-attempts cap, discount cap, or contact-hours window. Like Disputed,
it always forces `escalate` — but it does *not* force the root cause to
`disputed`; the diagnosis and confidence shown still reflect the real
Failure Code.

**Why this matters:** check Do Not Contact alone (Disputed left unchecked)
on a case that would otherwise get a nudge or incentive — the reasoning
chain still shows the real diagnosis and tier Strategy picked, but Guardrail
blocks it before Execution, which is a different story than the Disputed
case (whose diagnosis itself is overridden).

## Hour of Day (0–23)

The simulated clock hour used to check whether the case falls inside the
allowed contact window. This only matters for tiers that actually contact
the customer (nudge, incentivized_nudge) — `silent_retry` and `escalate`
never check it.

**Current window (live, from `guardrail.DefaultCaps` in `caps_config.go`,
not a fixed constant — this is the same Guardrail Policy shown on the
Guardrail Policy panel):** `ContactWindowStart = 9`, `ContactWindowEnd = 20`
— i.e. contact is allowed from 09:00 up to but not including 20:00.

Unlike Disputed and Do Not Contact, an out-of-window hour does **not**
escalate — Guardrail's `contact_hours` rule is the one rule that *holds* the
case for retry instead, keeping the tier Strategy chose.

**Why this matters:** pick any failure code that resolves to a
contact-requiring tier, set Hour of Day to 20, then to 19 — 20 is held
("outside contact hours, held for retry"), 19 is allowed. This is the only
field whose current value is fetched live rather than hardcoded, so it will
track any future policy change.

## Prior Contact Attempts

The count of contact attempts already made on this case before this
simulated run. Like Hour of Day, this only matters for tiers that contact
the customer.

**Current cap (live, from `guardrail.DefaultCaps`):** `MaxContactAttempts = 3`.
Guardrail's rule fires when `priorContactAttempts >= 3`, forcing `escalate`.

**Why this matters:** set Prior Contact Attempts to 2 vs. 3 on a case that
would otherwise get a nudge — 2 is allowed, 3 escalates ("cap reached").
This is the same rule that produces the two-strike promise-tracker
escalation you see on repeated-broken-promise cases in the batch dataset.
