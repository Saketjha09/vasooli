package diagnosis

import (
	"errors"
	"testing"

	"vasooli/internal/detector"
)

func TestClassify_FailureCodeMapping(t *testing.T) {
	cases := []struct {
		failureCode string
		wantRoot    string
	}{
		{"GATEWAY_TIMEOUT", RootCauseTransientGateway},
		{"BANK_SERVER_ERROR", RootCauseTransientGateway},
		{"CARD_EXPIRED", RootCauseCardExpired},
		{"INSUFFICIENT_FUNDS", RootCauseInsufficientFunds},
		{"CHECKOUT_ABANDONED", RootCauseCheckoutFriction},
		{"OTP_TIMEOUT", RootCauseCheckoutFriction},
		{"INVOICE_OVERDUE_NO_RESPONSE", RootCauseWillfulNonpayment},
		{"PAYMENT_DECLINED_REPEATED", RootCauseWillfulNonpayment},
	}

	for _, c := range cases {
		event := detector.Event{TransactionID: "t1", FailureCode: c.failureCode}
		got, err := Classify(event, "case-1")
		if err != nil {
			t.Fatalf("failure_code %s: unexpected error: %v", c.failureCode, err)
		}
		if got.RootCause != c.wantRoot {
			t.Errorf("failure_code %s: got root cause %s, want %s", c.failureCode, got.RootCause, c.wantRoot)
		}
		if got.Confidence <= 0 || got.Confidence > 1 {
			t.Errorf("failure_code %s: confidence %v out of (0,1] range", c.failureCode, got.Confidence)
		}
		if len(got.Evidence) == 0 {
			t.Errorf("failure_code %s: expected non-empty evidence", c.failureCode)
		}
		if got.CaseID != "case-1" {
			t.Errorf("failure_code %s: case ID not passed through, got %s", c.failureCode, got.CaseID)
		}
	}
}

func TestClassify_DisputedOverridesFailureCode(t *testing.T) {
	// Even a failure_code that would normally map to transient_gateway must
	// be overridden to "disputed" when the customer has an active dispute.
	event := detector.Event{TransactionID: "t2", FailureCode: "GATEWAY_TIMEOUT", DisputedFlag: true}

	got, err := Classify(event, "case-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.RootCause != RootCauseDisputed {
		t.Fatalf("expected disputed_flag to override failure_code mapping, got root cause %s", got.RootCause)
	}
	if got.Confidence != 0.99 {
		t.Errorf("expected disputed confidence 0.99, got %v", got.Confidence)
	}
}

func TestClassify_UnmappedFailureCodeReturnsError(t *testing.T) {
	event := detector.Event{TransactionID: "t3", FailureCode: "SOMETHING_UNKNOWN"}

	_, err := Classify(event, "case-3")
	if err == nil {
		t.Fatal("expected error for unmapped failure_code, got nil")
	}
	var unmapped ErrUnmappedFailureCode
	if !errors.As(err, &unmapped) {
		t.Fatalf("expected ErrUnmappedFailureCode, got %T: %v", err, err)
	}
	if unmapped.FailureCode != "SOMETHING_UNKNOWN" {
		t.Errorf("error carries wrong failure_code: %s", unmapped.FailureCode)
	}
}

func TestClassify_AllDemoFixtureFailureCodesAreMapped(t *testing.T) {
	// Guards against the demo dataset (backend/fixtures/demo_dataset.json)
	// drifting ahead of this rule table.
	demoCodes := []string{
		"GATEWAY_TIMEOUT", "BANK_SERVER_ERROR", "CARD_EXPIRED", "INSUFFICIENT_FUNDS",
		"CHECKOUT_ABANDONED", "OTP_TIMEOUT", "INVOICE_OVERDUE_NO_RESPONSE", "PAYMENT_DECLINED_REPEATED",
	}
	for _, code := range demoCodes {
		if _, ok := failureCodeMap[code]; !ok {
			t.Errorf("demo fixture uses failure_code %s but it has no rule mapping", code)
		}
	}
}
