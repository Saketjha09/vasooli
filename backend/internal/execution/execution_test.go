package execution

import (
	"errors"
	"testing"

	"vasooli/internal/detector"
	"vasooli/internal/guardrail"
	"vasooli/internal/strategy"
)

func TestSimulate_UnverifiedGuardrailResultReturnsError(t *testing.T) {
	// A zero-value GuardrailResult can't be forged into looking "allowed"
	// from outside the guardrail package (verified is unexported), but a
	// zero-value literal is still constructible — confirm Simulate rejects
	// it rather than treating it as a valid (and dangerously permissive)
	// result.
	var unverified guardrail.GuardrailResult
	decision := strategy.StrategyDecision{CaseID: "c0", Tier: strategy.TierNudge}
	event := detector.Event{TransactionID: "t0", Amount: 100}

	_, err := Simulate(unverified, decision, event, nil)
	if !errors.Is(err, ErrGuardrailNotVerified) {
		t.Fatalf("expected ErrGuardrailNotVerified, got %v", err)
	}
}

func TestSimulate_HeldCaseProducesNoFabricatedOutcome(t *testing.T) {
	decision := strategy.StrategyDecision{CaseID: "c1", Tier: strategy.TierNudge, Channel: strategy.ChannelEmail}
	event := detector.Event{TransactionID: "t1", Amount: 500, HistoryScore: 90}
	g := guardrail.Check(decision, event, guardrail.DefaultCaps, 3, 0) // 3am, outside contact window -> held

	got, err := Simulate(g, decision, event, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Result != OutcomeHeld {
		t.Fatalf("expected held outcome, got %s", got.Result)
	}
	if got.Amount != 0 {
		t.Errorf("held case must not report a recovered amount, got %v", got.Amount)
	}
}

func TestSimulate_BlockedEscalatedCase(t *testing.T) {
	decision := strategy.StrategyDecision{CaseID: "c2", Tier: strategy.TierNudge, Channel: strategy.ChannelEmail}
	event := detector.Event{TransactionID: "t2", Amount: 999, HistoryScore: 95, DisputedFlag: true}
	g := guardrail.Check(decision, event, guardrail.DefaultCaps, 12, 0)

	got, err := Simulate(g, decision, event, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Result != OutcomeEscalated {
		t.Fatalf("expected escalated outcome, got %s", got.Result)
	}
	if got.Amount != 0 {
		t.Errorf("blocked case must not report a recovered amount, got %v", got.Amount)
	}
}

func TestSimulate_NativeEscalateTier_NoRuleFired(t *testing.T) {
	// Strategy itself chose escalate (e.g. low-history willful_nonpayment);
	// guardrail finds nothing to block since escalate has no contact
	// channel, so RuleFired ends up empty.
	decision := strategy.StrategyDecision{CaseID: "c3", Tier: strategy.TierEscalate, Channel: strategy.ChannelNone}
	event := detector.Event{TransactionID: "t3", Amount: 100}
	g := guardrail.Check(decision, event, guardrail.DefaultCaps, 12, 0)
	if g.RuleFired != "" {
		t.Fatalf("test setup invalid: expected no rule to fire, got %s", g.RuleFired)
	}

	got, err := Simulate(g, decision, event, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Result != OutcomeEscalated {
		t.Fatalf("expected escalated outcome, got %s", got.Result)
	}
}

func TestSimulate_SilentRetryAlwaysRecovers(t *testing.T) {
	decision := strategy.StrategyDecision{CaseID: "c4", Tier: strategy.TierSilentRetry, Channel: strategy.ChannelNone}
	event := detector.Event{TransactionID: "t4", Amount: 1234.56, HistoryScore: 5} // low history irrelevant here
	g := guardrail.Check(decision, event, guardrail.DefaultCaps, 12, 0)

	got, err := Simulate(g, decision, event, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Result != OutcomeRecovered {
		t.Fatalf("expected recovered for silent_retry, got %s", got.Result)
	}
	if got.Amount != 1234.56 {
		t.Errorf("expected full amount recovered, got %v", got.Amount)
	}
}

func TestSimulate_ContactedTier_HistoryScoreBuckets(t *testing.T) {
	cases := []struct {
		name         string
		historyScore int
		wantResult   string
	}{
		{"at_recovered_threshold", 70, OutcomeRecovered},
		{"above_recovered_threshold", 95, OutcomeRecovered},
		{"mid_bucket_no_response", 55, OutcomeNoResponse},
		{"just_below_recovered", 69, OutcomeNoResponse},
		{"at_broken_promise_boundary", 40, OutcomeNoResponse},
		{"below_broken_promise_threshold", 39, OutcomeBrokenPromise},
		{"very_low_history", 5, OutcomeBrokenPromise},
	}

	for _, c := range cases {
		decision := strategy.StrategyDecision{CaseID: "c5", Tier: strategy.TierNudge, Channel: strategy.ChannelEmail}
		event := detector.Event{TransactionID: "t5", Amount: 1000, HistoryScore: c.historyScore}
		g := guardrail.Check(decision, event, guardrail.DefaultCaps, 12, 0)

		got, err := Simulate(g, decision, event, nil)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if got.Result != c.wantResult {
			t.Errorf("%s: historyScore=%d got %s, want %s", c.name, c.historyScore, got.Result, c.wantResult)
		}
		if c.wantResult == OutcomeRecovered && got.Amount != 1000 {
			t.Errorf("%s: expected full amount for nudge (no discount), got %v", c.name, got.Amount)
		}
		if c.wantResult != OutcomeRecovered && got.Amount != 0 {
			t.Errorf("%s: expected 0 amount for non-recovered outcome, got %v", c.name, got.Amount)
		}
	}
}

func TestSimulate_IncentivizedNudge_AppliesDiscountOnRecovery(t *testing.T) {
	decision := strategy.StrategyDecision{CaseID: "c6", Tier: strategy.TierIncentivizedNudge, Channel: strategy.ChannelEmail, DiscountPct: 8.0}
	event := detector.Event{TransactionID: "t6", Amount: 1000, HistoryScore: 90} // recovered bucket
	g := guardrail.Check(decision, event, guardrail.DefaultCaps, 12, 0)

	got, err := Simulate(g, decision, event, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Result != OutcomeRecovered {
		t.Fatalf("expected recovered, got %s", got.Result)
	}
	if got.Amount != 920 {
		t.Errorf("expected 8%% discount applied (920), got %v", got.Amount)
	}
}

func TestSimulate_ScenarioOverride_ForcesBrokenPromise(t *testing.T) {
	decision := strategy.StrategyDecision{CaseID: "c7", Tier: strategy.TierNudge, Channel: strategy.ChannelEmail}
	event := detector.Event{TransactionID: "t7", Amount: 500, HistoryScore: 95} // would normally be "recovered"
	g := guardrail.Check(decision, event, guardrail.DefaultCaps, 12, 0)
	scenarios := map[string]string{"t7": ScenarioRepeatedBrokenPromise}

	got, err := Simulate(g, decision, event, scenarios)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Result != OutcomeBrokenPromise {
		t.Fatalf("expected scripted scenario override to force broken_promise regardless of historyScore, got %s", got.Result)
	}
	if got.Amount != 0 {
		t.Errorf("expected 0 amount for broken_promise, got %v", got.Amount)
	}
}

func TestLoadScenarios_RealFixtureFile(t *testing.T) {
	scenarios, err := LoadScenarios("../../fixtures/demo_dataset.json")
	if err != nil {
		t.Fatalf("unexpected error loading real fixture: %v", err)
	}

	count := 0
	foundRepeatedBrokenPromise := false
	for _, s := range scenarios {
		count++
		if s == ScenarioRepeatedBrokenPromise {
			foundRepeatedBrokenPromise = true
		}
	}
	if count != 2 {
		t.Errorf("expected exactly 2 tagged scenarios (repeated_broken_promise + disputed_showcase), got %d: %v", count, scenarios)
	}
	if !foundRepeatedBrokenPromise {
		t.Error("expected the repeated_broken_promise scenario tag to be present in the real fixture")
	}
}
