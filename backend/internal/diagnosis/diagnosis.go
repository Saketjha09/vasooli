// Package diagnosis implements the Diagnosis Agent (MRD 4.2, PRD FR2): a
// deterministic, rule-based classifier from failure_code (+ dispute override)
// to exactly one root cause, with a fixed confidence and an evidence trail.
// No LLM call here by design — outcomes must be identical every demo run.
package diagnosis

import (
	"fmt"

	"vasooli/internal/detector"
)

// ErrUnmappedFailureCode is returned when an event's failure_code has no
// entry in failureCodeMap. The demo dataset only uses the 8 mapped codes;
// surfacing this as an error (rather than guessing a root cause) keeps the
// classifier from inventing an undocumented decision path.
type ErrUnmappedFailureCode struct {
	FailureCode string
}

func (e ErrUnmappedFailureCode) Error() string {
	return fmt.Sprintf("diagnosis: no rule mapped for failure_code %q", e.FailureCode)
}

const (
	RootCauseTransientGateway  = "transient_gateway"
	RootCauseCardExpired       = "card_expired"
	RootCauseInsufficientFunds = "insufficient_funds"
	RootCauseCheckoutFriction  = "checkout_friction"
	RootCauseWillfulNonpayment = "willful_nonpayment"
	RootCauseDisputed          = "disputed"
)

// DiagnosisResult matches the struct contract in the technical architecture
// doc section 3.
type DiagnosisResult struct {
	CaseID     string
	RootCause  string
	Confidence float64
	Evidence   []string
}

// failureCodeMap is the rule table: failure_code -> (root cause, confidence).
// Kept as a single lookup table so the mapping is inspectable at a glance,
// per PRD FR3's "no undocumented judgment calls" bar.
var failureCodeMap = map[string]struct {
	rootCause  string
	confidence float64
}{
	"GATEWAY_TIMEOUT":             {RootCauseTransientGateway, 0.95},
	"BANK_SERVER_ERROR":           {RootCauseTransientGateway, 0.95},
	"CARD_EXPIRED":                {RootCauseCardExpired, 0.95},
	"INSUFFICIENT_FUNDS":          {RootCauseInsufficientFunds, 0.95},
	"CHECKOUT_ABANDONED":          {RootCauseCheckoutFriction, 0.9},
	"OTP_TIMEOUT":                 {RootCauseCheckoutFriction, 0.9},
	"INVOICE_OVERDUE_NO_RESPONSE": {RootCauseWillfulNonpayment, 0.85},
	"PAYMENT_DECLINED_REPEATED":   {RootCauseWillfulNonpayment, 0.85},
}

// Classify assigns exactly one root cause + confidence + evidence to an
// event. caseID is supplied by the caller (the pipeline orchestrator owns
// case-row creation) rather than carried on Event itself.
//
// Returns ErrUnmappedFailureCode if event.FailureCode isn't in failureCodeMap
// and the event isn't disputed — callers should treat this as a data/config
// gap, not silently classify it.
func Classify(event detector.Event, caseID string) (DiagnosisResult, error) {
	if event.DisputedFlag {
		return DiagnosisResult{
			CaseID:     caseID,
			RootCause:  RootCauseDisputed,
			Confidence: 0.99,
			Evidence:   []string{"customer.disputed_flag=true"},
		}, nil
	}

	rule, ok := failureCodeMap[event.FailureCode]
	if !ok {
		return DiagnosisResult{}, ErrUnmappedFailureCode{FailureCode: event.FailureCode}
	}

	return DiagnosisResult{
		CaseID:     caseID,
		RootCause:  rule.rootCause,
		Confidence: rule.confidence,
		Evidence:   []string{fmt.Sprintf("failure_code=%s", event.FailureCode)},
	}, nil
}
