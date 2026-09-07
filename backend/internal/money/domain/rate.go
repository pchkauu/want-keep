package domain

import "errors"

var ErrInvalidRate = errors.New("invalid rate")

// Rate is a supplied finite decimal observation, in quote units per one base unit.
// Cross-rate calculation belongs to valuation and must retain its source legs.
type Rate struct {
	base  Asset
	quote Asset
	value Money
}

func NewRate(base, quote Asset, value string) (Rate, error) {
	if _, err := ParseAsset(string(base)); err != nil {
		return Rate{}, err
	}
	amount, err := NewMoney(value, quote)
	if err != nil {
		return Rate{}, err
	}
	if base == quote || amount.Sign() <= 0 {
		return Rate{}, ErrInvalidRate
	}
	return Rate{base: base, quote: quote, value: amount}, nil
}

func (r Rate) Base() Asset   { return r.base }
func (r Rate) Quote() Asset  { return r.quote }
func (r Rate) Value() string { return r.value.Amount() }
func (r Rate) Validate() error {
	_, err := NewRate(r.base, r.quote, r.Value())
	return err
}
