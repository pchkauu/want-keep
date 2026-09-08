package domain_test

import (
	"errors"
	"testing"

	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
)

func TestReservationAndActualCostAreExact(t *testing.T) {
	pricing := ai.TerraPricing()
	reserved, err := pricing.Reservation(1000, 2048)
	if err != nil || reserved.String() != "0.027156" {
		t.Fatalf("reservation = %s, %v", reserved.String(), err)
	}
	writes := int64(100)
	actual, conservative, err := pricing.Actual(ai.Usage{InputTokens: 1000, CachedTokens: 200, CacheWriteTokens: &writes, OutputTokens: 500, ReasoningTokens: 300})
	if err != nil || conservative || actual.String() != "0.00769" {
		t.Fatalf("actual = %s, conservative=%v, err=%v", actual.String(), conservative, err)
	}
	actual, conservative, err = pricing.Actual(ai.Usage{InputTokens: 1000, CachedTokens: 200, OutputTokens: 500, ReasoningTokens: 300})
	if err != nil || !conservative || actual.String() != "0.00804" {
		t.Fatalf("conservative actual = %s, conservative=%v, err=%v", actual.String(), conservative, err)
	}
}

func TestUsageRejectsContradictions(t *testing.T) {
	writes := int64(801)
	for _, usage := range []ai.Usage{
		{InputTokens: -1},
		{InputTokens: 10, CachedTokens: 11},
		{InputTokens: 1000, CachedTokens: 200, CacheWriteTokens: &writes},
		{OutputTokens: 2, ReasoningTokens: 3},
	} {
		if _, _, err := ai.TerraPricing().Actual(usage); !errors.Is(err, ai.ErrInvalidUsage) {
			t.Fatalf("usage accepted: %+v (%v)", usage, err)
		}
	}
}

func TestCostNeverUsesFloatOrNegativeValues(t *testing.T) {
	left, _ := ai.NewCost("49.999999")
	right, _ := ai.NewCost("0.000001")
	total, err := left.Add(right)
	if err != nil || total.String() != "50" {
		t.Fatalf("total = %s, %v", total.String(), err)
	}
	if _, err = right.Subtract(total); !errors.Is(err, ai.ErrInvalidCost) {
		t.Fatal("negative cost accepted")
	}
}

func TestCostPreservesSupportedInputPrecision(t *testing.T) {
	left := ai.MustCost("0.000000000000000000123400")
	right := ai.MustCost("0.000000000000000000000006")
	if left.String() != "0.0000000000000000001234" {
		t.Fatalf("constructor rounded exact input: %s", left.String())
	}
	total, err := left.Add(right)
	if err != nil || total.String() != "0.000000000000000000123406" {
		t.Fatalf("exact high-precision addition: %s, %v", total.String(), err)
	}
}
