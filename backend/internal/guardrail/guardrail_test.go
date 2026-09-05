package guardrail

import (
	"strings"
	"testing"

	"vasooli/internal/detector"
	"vasooli/internal/strategy"
)

// TestCheck_DisputedCase_ZeroLeakage is the explicit test for the single most
// important behavior in the project: a disputed case must be blocked and
// escalated, with no automated contact, no matter what tier Strategy chose
// or what other flags/attempts/hours look like.
func TestCheck_DisputedCase_ZeroLeakage(t *testing.T) {
	disputedEvent := detector.Event{CustomerID: "c-disputed", DisputedFlag: true, DoNotContact: false}

	tiersToTry := []strategy.StrategyDecision{
		{CaseID: "case-1", Tier: strategy.TierNudge, Channel: strategy.ChannelEmail},
		{CaseID: "case-1", Tier: strategy.TierIncentivizedNudge, Channel: strategy.ChannelEmail, DiscountPct: 8.0},
		{CaseID: "case-1", Tier: strategy.TierEscalate, Channel: strategy.ChannelNone},
		{CaseID: "case-1", Tier: strategy.TierSilentRetry, Channel: strategy.ChannelNone},
	}

	// Also vary simulated hour and prior attempts to prove the dispute check
	// wins regardless — it must be evaluated first, unconditionally.
	for _, decision := range tiersToTry {
		for _, hour := range []int{3, 12, 22} { // outside, inside, outside window
			for _, attempts := range []int{0, 3, 5} { // under, at, over cap
				got := Check(decision, disputedEvent, DefaultCaps, hour, attempts)

				if got.Allowed {
					t.Fatalf("tier=%s hour=%d attempts=%d: disputed case must never be Allowed, got Allowed=true", decision.Tier, hour, attempts)
				}
				if got.FinalTier != strategy.TierEscalate {
					t.Fatalf("tier=%s hour=%d attempts=%d: disputed case must escalate, got FinalTier=%s", decision.Tier, hour, attempts, got.FinalTier)
				}
				if got.RuleFired != RuleDisputeFlag {
					t.Fatalf("tier=%s hour=%d attempts=%d: expected rule_fired=%s, got %s", decision.Tier, hour, attempts, RuleDisputeFlag, got.RuleFired)
				}
				if got.Held {
					t.Fatalf("tier=%s hour=%d attempts=%d: disputed case must be a hard block, not a hold", decision.Tier, hour, attempts)
				}
			}
		}
	}
}

func TestCheck_DoNotContact_BlocksAndEscalates(t *testing.T) {
	event := detector.Event{DisputedFlag: false, DoNotContact: true}
	decision := strategy.StrategyDecision{CaseID: "case-2", Tier: strategy.TierNudge, Channel: strategy.ChannelEmail}

	got := Check(decision, event, DefaultCaps, 12, 0)
	if got.Allowed {
		t.Fatal("expected do_not_contact to block")
	}
	if got.FinalTier != strategy.TierEscalate {
		t.Errorf("expected escalate, got %s", got.FinalTier)
	}
	if got.RuleFired != RuleDoNotContact {
		t.Errorf("expected rule %s, got %s", RuleDoNotContact, got.RuleFired)
	}
}

func TestCheck_MaxContactAttempts_BlocksAtCap(t *testing.T) {
	event := detector.Event{}
	decision := strategy.StrategyDecision{CaseID: "case-3", Tier: strategy.TierNudge, Channel: strategy.ChannelEmail}

	got := Check(decision, event, DefaultCaps, 12, DefaultCaps.MaxContactAttempts)
	if got.Allowed {
		t.Fatal("expected max_contact_attempts to block at cap")
	}
	if got.RuleFired != RuleMaxContactAttempts {
		t.Errorf("expected rule %s, got %s", RuleMaxContactAttempts, got.RuleFired)
	}
	if got.FinalTier != strategy.TierEscalate {
		t.Errorf("expected escalate, got %s", got.FinalTier)
	}
}

func TestCheck_MaxContactAttempts_DoesNotApplyToSilentRetry(t *testing.T) {
	event := detector.Event{}
	decision := strategy.StrategyDecision{CaseID: "case-3b", Tier: strategy.TierSilentRetry, Channel: strategy.ChannelNone}

	got := Check(decision, event, DefaultCaps, 12, 99) // way over cap, but no contact involved
	if !got.Allowed {
		t.Fatalf("silent_retry has no contact channel, contact-attempt cap should not apply, got blocked: %s", got.RuleFired)
	}
}

func TestCheck_MaxDiscountPct_AllowsWithinCap(t *testing.T) {
	event := detector.Event{}
	decision := strategy.StrategyDecision{CaseID: "case-4", Tier: strategy.TierIncentivizedNudge, Channel: strategy.ChannelEmail, DiscountPct: 8.0}

	got := Check(decision, event, DefaultCaps, 12, 0)
	if !got.Allowed {
		t.Fatalf("expected 8%% discount under 10%% cap to be allowed, got blocked: %s", got.RuleFired)
	}

	evidenceJoined := strings.Join(got.Evidence, " | ")
	if !strings.Contains(evidenceJoined, "discountPct=8.0") || !strings.Contains(evidenceJoined, "cap=10.0") {
		t.Errorf("expected evidence to show actual discount checked against cap, got: %s", evidenceJoined)
	}
}

func TestCheck_MaxDiscountPct_BlocksOverCap(t *testing.T) {
	event := detector.Event{}
	decision := strategy.StrategyDecision{CaseID: "case-5", Tier: strategy.TierIncentivizedNudge, Channel: strategy.ChannelEmail, DiscountPct: 15.0}

	got := Check(decision, event, DefaultCaps, 12, 0)
	if got.Allowed {
		t.Fatal("expected discount over cap to block")
	}
	if got.RuleFired != RuleMaxDiscountPct {
		t.Errorf("expected rule %s, got %s", RuleMaxDiscountPct, got.RuleFired)
	}
	if got.FinalTier != strategy.TierEscalate {
		t.Errorf("expected escalate, got %s", got.FinalTier)
	}
}

func TestCheck_ContactHours_HoldsWithoutEscalating(t *testing.T) {
	event := detector.Event{}
	decision := strategy.StrategyDecision{CaseID: "case-6", Tier: strategy.TierNudge, Channel: strategy.ChannelEmail}

	got := Check(decision, event, DefaultCaps, 3, 0) // 3am, outside 9-20 window
	if got.Allowed {
		t.Fatal("expected outside-hours contact to be held, not allowed")
	}
	if !got.Held {
		t.Fatal("expected Held=true for contact-hours rule")
	}
	if got.FinalTier != strategy.TierNudge {
		t.Errorf("contact-hours hold must NOT escalate the tier, got FinalTier=%s", got.FinalTier)
	}
	if got.RuleFired != RuleContactHours {
		t.Errorf("expected rule %s, got %s", RuleContactHours, got.RuleFired)
	}
}

func TestCheck_ContactHours_DoesNotApplyToSilentRetry(t *testing.T) {
	event := detector.Event{}
	decision := strategy.StrategyDecision{CaseID: "case-6b", Tier: strategy.TierSilentRetry, Channel: strategy.ChannelNone}

	got := Check(decision, event, DefaultCaps, 3, 0)
	if !got.Allowed {
		t.Fatalf("silent_retry has no contact, contact-hours rule should not apply, got held/blocked: %s", got.RuleFired)
	}
}

func TestCheck_AllowsCompliantCase(t *testing.T) {
	event := detector.Event{DisputedFlag: false, DoNotContact: false}
	decision := strategy.StrategyDecision{CaseID: "case-7", Tier: strategy.TierNudge, Channel: strategy.ChannelEmail}

	got := Check(decision, event, DefaultCaps, 12, 1)
	if !got.Allowed {
		t.Fatalf("expected compliant case to be allowed, got blocked: %s (%s)", got.RuleFired, got.Reason)
	}
	if got.FinalTier != strategy.TierNudge {
		t.Errorf("expected FinalTier unchanged from decision.Tier, got %s", got.FinalTier)
	}
	if got.RuleFired != "" {
		t.Errorf("expected no rule fired for compliant case, got %s", got.RuleFired)
	}
}
