-- Vasooli core data model (MRD section 5).
-- created_at columns on cases/policy_events/promises are bookkeeping additions
-- beyond the literal MRD field list, added for audit ordering only.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE customers (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name            text NOT NULL,
    history_score   integer NOT NULL,
    disputed_flag   boolean NOT NULL DEFAULT false,
    do_not_contact  boolean NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE transactions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id     uuid NOT NULL,
    customer_id     uuid NOT NULL REFERENCES customers(id),
    amount          numeric(12, 2) NOT NULL,
    status          text NOT NULL,
    failure_code    text NOT NULL,
    created_at      timestamptz NOT NULL
);

CREATE INDEX idx_transactions_customer_id ON transactions(customer_id);

CREATE TABLE cases (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id  uuid NOT NULL REFERENCES transactions(id),
    root_cause      text,
    confidence      numeric(3, 2),
    tier_chosen     text,
    status          text NOT NULL DEFAULT 'open',
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_cases_transaction_id ON cases(transaction_id);

CREATE TABLE policy_events (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id         uuid NOT NULL REFERENCES cases(id),
    rule_fired      text,
    action_blocked  boolean NOT NULL DEFAULT false,
    reason          text,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_policy_events_case_id ON policy_events(case_id);

CREATE TABLE promises (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id         uuid NOT NULL REFERENCES cases(id),
    promised_date   date NOT NULL,
    kept            boolean,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_promises_case_id ON promises(case_id);

CREATE TABLE audit_log (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id             uuid NOT NULL REFERENCES cases(id),
    agent_name          text NOT NULL,
    decision            text,
    confidence          numeric(3, 2),
    alternatives_json   jsonb,
    timestamp           timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_case_id ON audit_log(case_id);
