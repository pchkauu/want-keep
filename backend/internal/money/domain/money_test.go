package domain_test

import (
	"errors"
	"strings"
	"testing"

	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestMoneyPrecisionAndInvalidInput(t *testing.T) {
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		for _, amount := range []string{"0", "0.000000000000000001", "12.000000000123", "-123.123456789123456789", strings.Repeat("9", 256)} {
			value, err := money.NewMoney(amount, asset)
			if err != nil || value.Amount() != amount || value.Asset() != asset {
				t.Fatalf("%s %s: %v", asset, amount, err)
			}
		}
	}
	for _, amount := range []string{"", "+1", "01", "1.", ".1", " 1", "1 ", "1e2", "NaN", "Infinity", "1,2", "١", strings.Repeat("9", 257)} {
		if _, err := money.NewMoney(amount, money.RUB); !errors.Is(err, money.ErrInvalidMoney) {
			t.Errorf("accepted invalid amount %q: %v", amount, err)
		}
	}
	for _, asset := range []money.Asset{"", "USDC.E", "RUR", "usd"} {
		if _, err := money.NewMoney("1", asset); !errors.Is(err, money.ErrUnsupportedAsset) {
			t.Errorf("accepted asset %s", asset)
		}
	}
}

func TestArithmeticPreservesOperandsAndAsset(t *testing.T) {
	a, _ := money.NewMoney("0.1", money.BTC)
	b, _ := money.NewMoney("0.2", money.BTC)
	sum, err := a.Add(b)
	if err != nil || sum.Amount() != "0.3" {
		t.Fatalf("sum: %s %v", sum.Amount(), err)
	}
	difference, err := sum.Subtract(b)
	if err != nil {
		t.Fatal(err)
	}
	if cmp, err := a.Compare(difference); err != nil || cmp != 0 {
		t.Fatalf("inverse: %v %d", err, cmp)
	}
	if a.Amount() != "0.1" || b.Amount() != "0.2" {
		t.Fatal("operands mutated")
	}
	usd, _ := money.NewMoney("0.2", money.USD)
	if _, err := a.Add(usd); !errors.Is(err, money.ErrAssetMismatch) {
		t.Fatal("mixed assets added")
	}
	if _, err := a.Subtract(usd); !errors.Is(err, money.ErrAssetMismatch) {
		t.Fatal("mixed assets subtracted")
	}
	if _, err := a.Compare(usd); !errors.Is(err, money.ErrAssetMismatch) {
		t.Fatal("mixed assets compared")
	}
	if _, err := (money.Money{}).Add(a); err == nil {
		t.Fatal("zero-value money accepted")
	}
	large, _ := money.NewMoney(strings.Repeat("9", 256), money.USD)
	one, _ := money.NewMoney("1", money.USD)
	if _, err := large.Add(one); !errors.Is(err, money.ErrInvalidMoney) {
		t.Fatal("oversized result accepted")
	}
}

func TestExplicitRounding(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		mode        money.Rounding
	}{{"1.235", "1.24", money.HalfEven}, {"1.245", "1.24", money.HalfEven}, {"-1.231", "-1.24", money.Floor}, {"1.239", "1.23", money.Floor}} {
		value, _ := money.NewMoney(tc.input, money.USDT)
		rounded, err := value.Round(2, tc.mode)
		if err != nil || rounded.Amount() != tc.want || value.Amount() != tc.input {
			t.Errorf("round %s: %s %v", tc.input, rounded.Amount(), err)
		}
	}
	value, _ := money.NewMoney("1.1", money.ETH)
	for _, scale := range []int32{-1, 255} {
		if _, err := value.Round(scale, money.Floor); err == nil {
			t.Fatal("invalid scale accepted")
		}
	}
	if _, err := value.Round(2, "implicit"); err == nil {
		t.Fatal("implicit rounding accepted")
	}
}

func TestLargestRemainderAllocation(t *testing.T) {
	for _, input := range []string{"10.01", "-10.01", "0.00", "0.000000000000000001"} {
		scale := int32(2)
		if strings.Contains(input, "000000001") {
			scale = 18
		}
		value, _ := money.NewMoney(input, money.ETH)
		weights := []money.Weight{{ID: "b", Value: "1"}, {ID: "a", Value: "1"}, {ID: "zero", Value: "0"}}
		result, err := value.Allocate(weights, scale)
		if err != nil {
			t.Fatal(err)
		}
		sum, _ := money.NewMoney("0", money.ETH)
		for _, part := range result {
			sum, err = sum.Add(part.Money)
			if err != nil {
				t.Fatal(err)
			}
		}
		if cmp, err := sum.Compare(value); err != nil || cmp != 0 {
			t.Fatal("allocation lost value")
		}
		if result[2].Money.Sign() != 0 {
			t.Fatal("zero weight got units")
		}
		reordered, err := value.Allocate([]money.Weight{weights[1], weights[0], weights[2]}, scale)
		if err != nil || reordered[0].Money.Amount() != result[1].Money.Amount() {
			t.Fatal("tie depends on input order")
		}
		if input == "10.01" && result[1].Money.Amount() != "5.01" {
			t.Fatal("stable ID did not receive remainder")
		}
	}
	value, _ := money.NewMoney("1.001", money.RUB)
	if _, err := value.Allocate([]money.Weight{{ID: "a", Value: "1"}}, 2); err == nil {
		t.Fatal("total silently rounded")
	}
	for _, weights := range [][]money.Weight{nil, {{ID: "a", Value: "0"}}, {{ID: "a", Value: "1"}, {ID: "a", Value: "1"}}, {{ID: "a", Value: "-1"}}, {{ID: "", Value: "1"}}, {{ID: "a", Value: "NaN"}}} {
		if _, err := value.Allocate(weights, 3); err == nil {
			t.Fatal("invalid allocation accepted")
		}
	}
}

func TestRateValidation(t *testing.T) {
	rate, err := money.NewRate(money.BTC, money.USD, "123.000000000123")
	if err != nil || rate.Value() != "123.000000000123" || rate.Base() != money.BTC {
		t.Fatal(err)
	}
	for _, value := range []string{"0", "-1", "NaN", "1e3"} {
		if _, err := money.NewRate(money.USDT, money.USD, value); err == nil {
			t.Fatal("invalid rate accepted")
		}
	}
	if _, err := money.NewRate(money.USD, money.USD, "1"); err == nil {
		t.Fatal("same asset rate accepted")
	}
}

func FuzzMoneyRoundTrip(f *testing.F) {
	for _, seed := range []string{"1.000000000001", "-0.00", "0", "1e5"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		value, err := money.NewMoney(input, money.BTC)
		if err != nil {
			return
		}
		restored, err := money.NewMoney(value.Amount(), value.Asset())
		if err != nil {
			t.Fatal(err)
		}
		if cmp, err := value.Compare(restored); err != nil || cmp != 0 {
			t.Fatal("round-trip changed money")
		}
	})
}
