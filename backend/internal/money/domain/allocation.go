package domain

import (
	"errors"
	"math/big"
	"sort"
)

var ErrInvalidAllocation = errors.New("invalid allocation")

type Weight struct {
	ID    string
	Value string
}

type Allocation struct {
	ID    string
	Money Money
}

// Allocate distributes exact quantum units. It never rounds the total to fit the requested scale.
func (m Money) Allocate(weights []Weight, scale int32) ([]Allocation, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if len(weights) == 0 || len(weights) > 1000 || scale < 0 || scale > MaxDecimalLength-2 {
		return nil, ErrInvalidAllocation
	}
	total, ok := new(big.Rat).SetString(m.Amount())
	if !ok {
		return nil, ErrInvalidAllocation
	}
	total.Abs(total)
	quantum := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	total.Mul(total, new(big.Rat).SetInt(quantum))
	if !total.IsInt() {
		return nil, ErrInvalidAllocation
	}
	values := make([]*big.Rat, len(weights))
	sum := new(big.Rat)
	seen := make(map[string]bool, len(weights))
	for i, weight := range weights {
		if weight.ID == "" || seen[weight.ID] || len(weight.Value) > MaxDecimalLength || !decimalSyntax.MatchString(weight.Value) {
			return nil, ErrInvalidAllocation
		}
		seen[weight.ID] = true
		value, valid := new(big.Rat).SetString(weight.Value)
		if !valid || value.Sign() < 0 {
			return nil, ErrInvalidAllocation
		}
		values[i] = value
		sum.Add(sum, value)
	}
	if sum.Sign() == 0 {
		return nil, ErrInvalidAllocation
	}
	units := make([]*big.Int, len(weights))
	remainders := make([]*big.Rat, len(weights))
	order := make([]int, len(weights))
	remaining := new(big.Int).Set(total.Num())
	for i, value := range values {
		share := new(big.Rat).Mul(total, value)
		share.Quo(share, sum)
		units[i] = new(big.Int).Quo(share.Num(), share.Denom())
		remainders[i] = new(big.Rat).Sub(share, new(big.Rat).SetInt(units[i]))
		remaining.Sub(remaining, units[i])
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool {
		left, right := order[i], order[j]
		if cmp := remainders[left].Cmp(remainders[right]); cmp != 0 {
			return cmp > 0
		}
		return weights[left].ID < weights[right].ID
	})
	if !remaining.IsInt64() || remaining.Sign() < 0 || remaining.Int64() >= int64(len(weights)) {
		return nil, ErrInvalidAllocation
	}
	for i := int64(0); i < remaining.Int64(); i++ {
		units[order[i]].Add(units[order[i]], big.NewInt(1))
	}
	result := make([]Allocation, len(weights))
	for i, value := range units {
		if m.Sign() < 0 {
			value.Neg(value)
		}
		amount := new(big.Rat).SetFrac(value, quantum).FloatString(int(scale))
		allocated, err := NewMoney(amount, m.asset)
		if err != nil {
			return nil, err
		}
		result[i] = Allocation{ID: weights[i].ID, Money: allocated}
	}
	return result, nil
}
