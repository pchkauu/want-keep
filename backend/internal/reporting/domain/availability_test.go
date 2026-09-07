package domain_test

import (
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"testing"
)

func TestKnownZeroIsDifferentFromMissing(t *testing.T) {
	zero, _ := money.NewMoney("0", money.RUB)
	known, err := reporting.KnownAmount(zero)
	if err != nil {
		t.Fatal(err)
	}
	if amount, ok := known.Value(); !ok || amount.Sign() != 0 {
		t.Fatal("known zero missing")
	}
	for _, state := range []reporting.Knowledge{reporting.Unknown, reporting.Unavailable} {
		missing, err := reporting.MissingAmount(state, "missing_rate")
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := missing.Value(); ok {
			t.Fatal("missing became zero")
		}
	}
	if _, err := reporting.MissingAmount(reporting.Known, "reason"); err == nil {
		t.Fatal("known without value")
	}
	if _, err := reporting.MissingAmount(reporting.Unknown, ""); err == nil {
		t.Fatal("missing reason")
	}
	reasons := []string{"history_gap"}
	coverage, err := reporting.NewCoverage(reporting.Partial, reasons)
	if err != nil {
		t.Fatal(err)
	}
	reasons[0] = "changed"
	copy := coverage.Reasons()
	copy[0] = "changed"
	if coverage.Reasons()[0] != "history_gap" {
		t.Fatal("mutable coverage leaked")
	}
	if _, err := reporting.NewCoverage(reporting.Complete, reasons); err == nil {
		t.Fatal("contradictory complete state")
	}
	if _, err := reporting.NewCoverage(reporting.Partial, nil); err == nil {
		t.Fatal("partial without reason")
	}
	if _, err := reporting.ParseFreshness("stale"); err != nil {
		t.Fatal(err)
	}
	if _, err := reporting.ParseFreshness("fresh-ish"); err == nil {
		t.Fatal("unknown freshness silently mapped")
	}
}
