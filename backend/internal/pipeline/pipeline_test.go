package pipeline

import (
	"testing"

	"vasooli/internal/detector"
	"vasooli/internal/execution"
	"vasooli/internal/guardrail"
	"vasooli/internal/promise"
	"vasooli/internal/strategy"
)

func TestStatusFor(t *testing.T) {
	cases := []struct {
		name       string
		outcome    execution.ExecutionOutcome
		promise    promise.PromiseResult
		wantStatus string
	}{
		{"recovered", execution.ExecutionOutcome{Result: execution.OutcomeRecovered}, promise.PromiseResult{}, "recovered"},
		{"held", execution.ExecutionOutcome{Result: execution.OutcomeHeld}, promise.PromiseResult{}, "held"},
		{"escalated_by_execution", execution.ExecutionOutcome{Result: execution.OutcomeEscalated}, promise.PromiseResult{}, "escalated"},
		{"escalated_by_promise", execution.ExecutionOutcome{Result: execution.OutcomeNoResponse}, promise.PromiseResult{Escalated: true}, "escalated"},
		{"no_response_open", execution.ExecutionOutcome{Result: execution.OutcomeNoResponse}, promise.PromiseResult{}, "open"},
		{"broken_promise_not_yet_escalated", execution.ExecutionOutcome{Result: execution.OutcomeBrokenPromise}, promise.PromiseResult{StrikeCount: 1}, "open"},
	}

	for _, c := range cases {
		got := statusFor(c.outcome, c.promise)
		if got != c.wantStatus {
			t.Errorf("%s: got status %s, want %s", c.name, got, c.wantStatus)
		}
	}
}

func TestGuardrailDecisionLabel(t *testing.T) {
	decision := strategy.StrategyDecision{CaseID: "c1", Tier: strategy.TierNudge, Channel: strategy.ChannelEmail}
	event := detector.Event{TransactionID: "t1"}
	disputedEvent := detector.Event{TransactionID: "t2", DisputedFlag: true}

	allowed := guardrail.Check(decision, event, guardrail.DefaultCaps, 12, 0)
	if got := guardrailDecisionLabel(allowed); got != "allowed tier=nudge" {
		t.Errorf("expected 'allowed tier=nudge', got %q", got)
	}

	held := guardrail.Check(decision, event, guardrail.DefaultCaps, 3, 0)
	if got := guardrailDecisionLabel(held); got != "held rule="+guardrail.RuleContactHours {
		t.Errorf("expected held label, got %q", got)
	}

	blocked := guardrail.Check(decision, disputedEvent, guardrail.DefaultCaps, 12, 0)
	wantBlocked := "blocked rule=" + guardrail.RuleDisputeFlag + " tier=" + strategy.TierEscalate
	if got := guardrailDecisionLabel(blocked); got != wantBlocked {
		t.Errorf("expected %q, got %q", wantBlocked, got)
	}
}
