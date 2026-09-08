package domain

import (
	"errors"
	"math/big"
	"regexp"
	"strings"
)

var (
	ErrInvalidCost  = errors.New("invalid AI cost")
	ErrInvalidUsage = errors.New("invalid AI usage")
	decimalPattern  = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)
)

// Cost is an exact non-negative USD amount. It deliberately has no float boundary.
type Cost struct{ value string }

func NewCost(value string) (Cost, error) {
	if len(value) == 0 || len(value) > 128 || !decimalPattern.MatchString(value) {
		return Cost{}, ErrInvalidCost
	}
	return Cost{value: canonicalInput(value)}, nil
}

func MustCost(value string) Cost {
	cost, err := NewCost(value)
	if err != nil {
		panic(err)
	}
	return cost
}

func (c Cost) String() string { return c.value }

func (c Cost) Validate() error {
	_, err := NewCost(c.value)
	return err
}

func (c Cost) Add(other Cost) (Cost, error) {
	left, right, err := c.ratPair(other)
	if err != nil {
		return Cost{}, err
	}
	return NewCost(decimalAtScale(new(big.Rat).Add(left, right), max(decimalScale(c.value), decimalScale(other.value))))
}

func (c Cost) Subtract(other Cost) (Cost, error) {
	left, right, err := c.ratPair(other)
	if err != nil {
		return Cost{}, err
	}
	result := new(big.Rat).Sub(left, right)
	if result.Sign() < 0 {
		return Cost{}, ErrInvalidCost
	}
	return NewCost(decimalAtScale(result, max(decimalScale(c.value), decimalScale(other.value))))
}

func (c Cost) Compare(other Cost) (int, error) {
	left, right, err := c.ratPair(other)
	if err != nil {
		return 0, err
	}
	return left.Cmp(right), nil
}

func (c Cost) ratPair(other Cost) (*big.Rat, *big.Rat, error) {
	if err := c.Validate(); err != nil {
		return nil, nil, err
	}
	if err := other.Validate(); err != nil {
		return nil, nil, err
	}
	left, _ := new(big.Rat).SetString(c.value)
	right, _ := new(big.Rat).SetString(other.value)
	return left, right, nil
}

func canonicalInput(value string) string {
	text := value
	if strings.Contains(text, ".") {
		text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	}
	if text == "" || text == "-0" {
		return "0"
	}
	return text
}

func decimalScale(value string) int {
	if point := strings.IndexByte(value, '.'); point >= 0 {
		return len(value) - point - 1
	}
	return 0
}

func decimalAtScale(value *big.Rat, scale int) string {
	return canonicalInput(value.FloatString(scale))
}

type Usage struct {
	InputTokens      int64
	CachedTokens     int64
	CacheWriteTokens *int64
	OutputTokens     int64
	ReasoningTokens  int64
}

func (u Usage) Validate() error {
	if u.InputTokens < 0 || u.CachedTokens < 0 || u.OutputTokens < 0 || u.ReasoningTokens < 0 || u.CachedTokens > u.InputTokens || u.ReasoningTokens > u.OutputTokens {
		return ErrInvalidUsage
	}
	if u.CacheWriteTokens != nil && (*u.CacheWriteTokens < 0 || *u.CacheWriteTokens > u.InputTokens-u.CachedTokens) {
		return ErrInvalidUsage
	}
	return nil
}

type Pricing struct {
	InputPerMillion, CachedPerMillion, CacheWritePerMillion, OutputPerMillion Cost
}

func TerraPricing() Pricing {
	return Pricing{
		InputPerMillion:      MustCost("2"),
		CachedPerMillion:     MustCost("0.2"),
		CacheWritePerMillion: MustCost("2.5"),
		OutputPerMillion:     MustCost("12"),
	}
}

func (p Pricing) Reservation(countedInput, maximumOutput int64) (Cost, error) {
	if countedInput < 0 || countedInput > 262144 || maximumOutput < 1 || maximumOutput > 8192 {
		return Cost{}, ErrInvalidUsage
	}
	return p.tokenCost(0, 0, countedInput+32, maximumOutput)
}

func (p Pricing) Actual(usage Usage) (Cost, bool, error) {
	if err := usage.Validate(); err != nil {
		return Cost{}, false, err
	}
	writes := usage.InputTokens - usage.CachedTokens
	conservative := usage.CacheWriteTokens == nil
	if usage.CacheWriteTokens != nil {
		writes = *usage.CacheWriteTokens
	}
	uncached := usage.InputTokens - usage.CachedTokens - writes
	cost, err := p.tokenCost(uncached, usage.CachedTokens, writes, usage.OutputTokens)
	return cost, conservative, err
}

func (p Pricing) tokenCost(input, cached, writes, output int64) (Cost, error) {
	if input < 0 || cached < 0 || writes < 0 || output < 0 {
		return Cost{}, ErrInvalidUsage
	}
	total := new(big.Rat)
	for _, item := range []struct {
		tokens int64
		price  Cost
	}{{input, p.InputPerMillion}, {cached, p.CachedPerMillion}, {writes, p.CacheWritePerMillion}, {output, p.OutputPerMillion}} {
		if err := item.price.Validate(); err != nil {
			return Cost{}, err
		}
		price, _ := new(big.Rat).SetString(item.price.String())
		total.Add(total, new(big.Rat).Mul(new(big.Rat).SetInt64(item.tokens), price))
	}
	total.Quo(total, new(big.Rat).SetInt64(1_000_000))
	maximumPriceScale := 0
	for _, price := range []Cost{p.InputPerMillion, p.CachedPerMillion, p.CacheWritePerMillion, p.OutputPerMillion} {
		maximumPriceScale = max(maximumPriceScale, decimalScale(price.String()))
	}
	return NewCost(decimalAtScale(total, maximumPriceScale+6))
}
