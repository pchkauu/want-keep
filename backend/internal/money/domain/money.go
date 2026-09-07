package domain

import (
	"errors"
	"regexp"

	"github.com/cockroachdb/apd/v3"
)

const MaxDecimalLength = 256

var (
	ErrInvalidMoney    = errors.New("invalid money")
	ErrAssetMismatch   = errors.New("asset mismatch")
	ErrInvalidRounding = errors.New("invalid rounding")
	decimalSyntax      = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)
)

type Money struct {
	amount apd.Decimal
	asset  Asset
}

type Rounding string

const (
	Floor    Rounding = "floor"
	HalfEven Rounding = "half_even"
)

func NewMoney(amount string, asset Asset) (Money, error) {
	if _, err := ParseAsset(string(asset)); err != nil {
		return Money{}, err
	}
	if len(amount) > MaxDecimalLength || !decimalSyntax.MatchString(amount) {
		return Money{}, ErrInvalidMoney
	}
	var value apd.Decimal
	if _, _, err := value.SetString(amount); err != nil {
		return Money{}, ErrInvalidMoney
	}
	if value.IsZero() {
		value.Negative = false
	}
	return Money{amount: value, asset: asset}, nil
}

func (m Money) Validate() error {
	if _, err := ParseAsset(string(m.asset)); err != nil {
		return err
	}
	if m.amount.Form != apd.Finite || len(m.Amount()) > MaxDecimalLength {
		return ErrInvalidMoney
	}
	return nil
}

func (m Money) Asset() Asset { return m.asset }

func (m Money) Amount() string { return m.amount.Text('f') }

func (m Money) Sign() int { return m.amount.Sign() }

func (m Money) Add(other Money) (Money, error) {
	if err := m.sameAsset(other); err != nil {
		return Money{}, err
	}
	context := apd.BaseContext
	var result apd.Decimal
	if _, err := context.Add(&result, &m.amount, &other.amount); err != nil {
		return Money{}, ErrInvalidMoney
	}
	return NewMoney(result.Text('f'), m.asset)
}

func (m Money) Subtract(other Money) (Money, error) {
	if err := m.sameAsset(other); err != nil {
		return Money{}, err
	}
	context := apd.BaseContext
	var result apd.Decimal
	if _, err := context.Sub(&result, &m.amount, &other.amount); err != nil {
		return Money{}, ErrInvalidMoney
	}
	return NewMoney(result.Text('f'), m.asset)
}

func (m Money) Compare(other Money) (int, error) {
	if err := m.sameAsset(other); err != nil {
		return 0, err
	}
	return m.amount.Cmp(&other.amount), nil
}

func (m Money) Round(scale int32, mode Rounding) (Money, error) {
	if err := m.Validate(); err != nil {
		return Money{}, err
	}
	if scale < 0 || scale > MaxDecimalLength-2 || (mode != Floor && mode != HalfEven) {
		return Money{}, ErrInvalidRounding
	}
	context := apd.BaseContext
	context.Precision = MaxDecimalLength * 2
	context.Rounding = apd.Rounder(mode)
	var result apd.Decimal
	if _, err := context.Quantize(&result, &m.amount, -scale); err != nil {
		return Money{}, ErrInvalidRounding
	}
	return NewMoney(result.Text('f'), m.asset)
}

func (m Money) sameAsset(other Money) error {
	if err := m.Validate(); err != nil {
		return err
	}
	if err := other.Validate(); err != nil {
		return err
	}
	if m.asset != other.asset {
		return ErrAssetMismatch
	}
	return nil
}
