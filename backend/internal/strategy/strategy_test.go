package strategy

import (
	"strings"
	"testing"

	"vasooli/internal/diagnosis"
)

func TestSelectTier_FixedMappings(t *testing.T) {
	cases := []struct {
		name        string
		rootCause   string
		wantTier    string
		wantChannel string
	}{
		{"disputed", diagnosis.RootCauseDisputed, TierEscalate, ChannelNone},
		{"transient_gateway", diagnosis.RootCauseTransientGateway, TierSilentRetry, ChannelNone},
		{"card_expired", diagnosis.RootCauseCardExpired, TierNudge, ChannelEmail},
		{"insufficient_funds", diagnosis.RootCauseInsufficientFunds, TierNudge, ChannelEmail},
		{"checkout_friction", diagnosis.RootCauseCheckoutFriction, TierNudge, ChannelSMS},
	}

	for _, c := range cases {
		d := diagnosis.DiagnosisResult{CaseID: "case-1", RootCause: c.rootCause}
		got := SelectTier(d, 50) // historyScore irrelevant for these root causes
		if got.Tier != c.wantTier {
			t.Errorf("%s: got tier %s, want %s", c.name, got.Tier, c.wantTier)
		}
		if got.Channel != c.wantChannel {
			t.Errorf("%s: got channel %s, want %s", c.name, got.Channel, c.wantChannel)
		}
		if len(got.Evidence) == 0 {
			t.Errorf("%s: expected non-empty evidence", c.name)
		}
		if got.CaseID != "case-1" {
			t.Errorf("%s: case ID not passed through", c.name)
		}
		if got.DiscountPct != 0 {
			t.Errorf("%s: expected no discount outside incentivized_nudge, got %.1f", c.name, got.DiscountPct)
		}
		if len(got.Alternatives) != 3 {
			t.Errorf("%s: expected exactly 3 rejected-alternative reasons (one per non-chosen tier), got %d: %v", c.name, len(got.Alternatives), got.Alternatives)
		}
		for _, alt := range got.Alternatives {
			if strings.HasPrefix(alt, got.Tier+":") {
				t.Errorf("%s: alternatives must not include the chosen tier itself, found %q", c.name, alt)
			}
		}
	}
}

func TestSelectTier_WillfulNonpayment_AtOrAboveThresholdGetsIncentive(t *testing.T) {
	d := diagnosis.DiagnosisResult{CaseID: "case-2", RootCause: diagnosis.RootCauseWillfulNonpayment}
	got := SelectTier(d, willfulNonpaymentHistoryThreshold) // exactly at threshold

	if got.Tier != TierIncentivizedNudge {
		t.Fatalf("expected incentivized_nudge at threshold, got %s", got.Tier)
	}
	if got.Channel != ChannelEmail {
		t.Errorf("expected email channel, got %s", got.Channel)
	}
	if got.DiscountPct != incentivizedNudgeDiscountPct {
		t.Errorf("expected discount %.1f, got %.1f", incentivizedNudgeDiscountPct, got.DiscountPct)
	}

	evidenceJoined := strings.Join(got.Evidence, " | ")
	if !strings.Contains(evidenceJoined, "historyScore=40") || !strings.Contains(evidenceJoined, "threshold=40") {
		t.Errorf("expected evidence to log actual historyScore and threshold, got: %s", evidenceJoined)
	}

	altsJoined := strings.Join(got.Alternatives, " | ")
	if !strings.Contains(altsJoined, "escalate:") || !strings.Contains(altsJoined, "nudge:") {
		t.Errorf("expected rejected-alternatives to cover escalate and nudge, got: %s", altsJoined)
	}
}

func TestSelectTier_WillfulNonpayment_BelowThresholdEscalates(t *testing.T) {
	d := diagnosis.DiagnosisResult{CaseID: "case-3", RootCause: diagnosis.RootCauseWillfulNonpayment}
	got := SelectTier(d, willfulNonpaymentHistoryThreshold-1)

	if got.Tier != TierEscalate {
		t.Fatalf("expected escalate below threshold, got %s", got.Tier)
	}
	if got.Channel != ChannelNone {
		t.Errorf("expected no channel for escalate, got %s", got.Channel)
	}

	evidenceJoined := strings.Join(got.Evidence, " | ")
	if !strings.Contains(evidenceJoined, "historyScore=39") {
		t.Errorf("expected evidence to log actual historyScore=39, got: %s", evidenceJoined)
	}

	altsJoined := strings.Join(got.Alternatives, " | ")
	if !strings.Contains(altsJoined, "incentivized_nudge:") || !strings.Contains(altsJoined, "historyScore=39") {
		t.Errorf("expected rejected-alternatives to explain incentivized_nudge was skipped using the actual historyScore, got: %s", altsJoined)
	}
}

func TestSelectTier_UnknownRootCausePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unrecognized root cause, got none")
		}
	}()
	SelectTier(diagnosis.DiagnosisResult{CaseID: "case-4", RootCause: "not_a_real_root_cause"}, 50)
}

func TestNudgeIgnoredUpgrade(t *testing.T) {
	got := NudgeIgnoredUpgrade("case-5")

	if got.CaseID != "case-5" {
		t.Errorf("expected case ID passed through, got %s", got.CaseID)
	}
	if got.Tier != TierIncentivizedNudge {
		t.Fatalf("expected upgrade to incentivized_nudge, got %s", got.Tier)
	}
	if got.Channel != ChannelEmail {
		t.Errorf("expected email channel, got %s", got.Channel)
	}
	if got.DiscountPct != incentivizedNudgeDiscountPct {
		t.Errorf("expected discount %.1f, got %.1f", incentivizedNudgeDiscountPct, got.DiscountPct)
	}
	if len(got.Evidence) == 0 {
		t.Error("expected non-empty evidence citing the prior no_response cycle")
	}
	if len(got.Alternatives) != 3 {
		t.Errorf("expected 3 rejected-alternative reasons, got %d: %v", len(got.Alternatives), got.Alternatives)
	}
}
