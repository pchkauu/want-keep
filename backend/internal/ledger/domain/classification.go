package domain

import (
	"errors"
	"math/big"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

var (
	ErrInvalidAllocation     = errors.New("invalid receipt allocation")
	ErrClarificationRequired = errors.New("receipt allocation needs clarification")
	quantitySyntax           = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)
)

type ReceiptItem struct {
	ID, Name, Quantity, CategoryID string
	Gross, Discount                money.Money
	Allocation                     AllocationSnapshot
}

func (i ReceiptItem) Validate() error {
	quantity, valid := new(big.Rat).SetString(i.Quantity)
	if i.ID == "" || strings.TrimSpace(i.Name) == "" || utf8.RuneCountInString(i.Name) > 2000 || len(i.Quantity) > money.MaxDecimalLength || !quantitySyntax.MatchString(i.Quantity) || !valid || quantity.Sign() <= 0 {
		return ErrInvalidAllocation
	}
	if err := i.Gross.Validate(); err != nil || i.Gross.Sign() <= 0 {
		return ErrInvalidAllocation
	}
	if err := i.Discount.Validate(); err != nil || i.Discount.Sign() < 0 || i.Discount.Asset() != i.Gross.Asset() {
		return ErrInvalidAllocation
	}
	if compared, err := i.Discount.Compare(i.Gross); err != nil || compared > 0 {
		return ErrInvalidAllocation
	}
	if i.Allocation.State != "" && i.Allocation.Validate() != nil {
		return ErrInvalidAllocation
	}
	return nil
}

func (i ReceiptItem) Net() (money.Money, error) { return i.Gross.Subtract(i.Discount) }

type ReceiptItemInput struct {
	ID, Name, Quantity, CategoryID string
	Gross                          money.Money
	Discount                       *money.Money
}

type ReceiptItemsCorrection struct {
	Clear         bool
	Items         []ReceiptItemInput
	TotalDiscount money.Money
}

type ClassificationProposal struct {
	CategoryID, MerchantID, MerchantAlias string
	ReceiptItems                          []ReceiptItem
}

func (p ClassificationProposal) ValidateFor(current Revision) error {
	if p.CategoryID == "" && p.MerchantID == "" && len(p.ReceiptItems) == 0 {
		return ErrInvalidAllocation
	}
	if p.MerchantAlias != "" && (p.MerchantID == "" || strings.TrimSpace(p.MerchantAlias) == "" || utf8.RuneCountInString(p.MerchantAlias) > 200) {
		return ErrInvalidAllocation
	}
	proposed := current.Clone()
	proposed.CategoryID = p.CategoryID
	proposed.MerchantID = p.MerchantID
	proposed.ReceiptItems = slices.Clone(p.ReceiptItems)
	return proposed.validateClassification()
}

func (c ReceiptItemsCorrection) Build(current Revision) ([]ReceiptItem, error) {
	if c.Clear {
		if len(c.Items) != 0 || c.TotalDiscount.Validate() == nil {
			return nil, ErrInvalidAllocation
		}
		return []ReceiptItem{}, nil
	}
	principal := current.rolePostings(Principal)
	if len(principal) != 1 || principal[0].Money.Sign() >= 0 {
		return nil, ErrInvalidAllocation
	}
	paid, err := money.NewMoney(strings.TrimPrefix(principal[0].Money.Amount(), "-"), principal[0].Money.Asset())
	if err != nil {
		return nil, ErrInvalidAllocation
	}
	return BuildReceiptItems(c.Items, c.TotalDiscount, paid)
}

func BuildReceiptItems(inputs []ReceiptItemInput, totalDiscount, paid money.Money) ([]ReceiptItem, error) {
	if len(inputs) < 1 || len(inputs) > 1000 || totalDiscount.Validate() != nil || paid.Validate() != nil || totalDiscount.Sign() < 0 || paid.Sign() <= 0 || totalDiscount.Asset() != paid.Asset() {
		return nil, ErrInvalidAllocation
	}
	seen := map[string]bool{}
	provided := 0
	weights := make([]money.Weight, 0, len(inputs))
	for _, input := range inputs {
		if input.ID == "" || seen[input.ID] || input.Gross.Validate() != nil || input.Gross.Sign() <= 0 || input.Gross.Asset() != paid.Asset() {
			return nil, ErrInvalidAllocation
		}
		seen[input.ID] = true
		weights = append(weights, money.Weight{ID: input.ID, Value: input.Gross.Amount()})
		if input.Discount != nil {
			provided++
		}
	}
	if provided != 0 && provided != len(inputs) {
		return nil, ErrClarificationRequired
	}
	discounts := map[string]money.Money{}
	if provided == 0 {
		allocated, err := totalDiscount.Allocate(weights, maxScale(appendMoney(inputs, totalDiscount, paid)))
		if err != nil {
			return nil, ErrInvalidAllocation
		}
		for _, allocation := range allocated {
			discounts[allocation.ID] = allocation.Money
		}
	} else {
		zero, _ := money.NewMoney("0", paid.Asset())
		sum := zero
		for _, input := range inputs {
			if input.Discount == nil || input.Discount.Asset() != paid.Asset() {
				return nil, ErrInvalidAllocation
			}
			var err error
			sum, err = sum.Add(*input.Discount)
			if err != nil {
				return nil, ErrInvalidAllocation
			}
			discounts[input.ID] = *input.Discount
		}
		if equal, _ := sum.Compare(totalDiscount); equal != 0 {
			return nil, ErrInvalidAllocation
		}
	}
	items := make([]ReceiptItem, 0, len(inputs))
	zero, _ := money.NewMoney("0", paid.Asset())
	netTotal := zero
	for _, input := range inputs {
		item := ReceiptItem{ID: input.ID, Name: strings.TrimSpace(input.Name), Quantity: input.Quantity, Gross: input.Gross, Discount: discounts[input.ID], CategoryID: input.CategoryID}
		if err := item.Validate(); err != nil {
			return nil, err
		}
		net, err := item.Net()
		if err != nil {
			return nil, ErrInvalidAllocation
		}
		netTotal, err = netTotal.Add(net)
		if err != nil {
			return nil, ErrInvalidAllocation
		}
		items = append(items, item)
	}
	if equal, _ := netTotal.Compare(paid); equal != 0 {
		return nil, ErrClarificationRequired
	}
	return items, nil
}

func appendMoney(inputs []ReceiptItemInput, values ...money.Money) []money.Money {
	result := slices.Clone(values)
	for _, input := range inputs {
		result = append(result, input.Gross)
		if input.Discount != nil {
			result = append(result, *input.Discount)
		}
	}
	return result
}

func maxScale(values []money.Money) int32 {
	var scale int
	for _, value := range values {
		if point := strings.IndexByte(value.Amount(), '.'); point >= 0 && len(value.Amount())-point-1 > scale {
			scale = len(value.Amount()) - point - 1
		}
	}
	return int32(scale)
}

func (r Revision) validateClassification() error {
	if (r.CategoryID != "" || r.MerchantID != "" || len(r.ReceiptItems) > 0) && r.Type != Expense {
		return ErrInvalidAllocation
	}
	if r.CategoryID != "" && len(r.ReceiptItems) > 0 {
		return ErrInvalidAllocation
	}
	if len(r.ReceiptItems) == 0 {
		return nil
	}
	principal := r.rolePostings(Principal)
	if len(principal) != 1 || principal[0].Money.Sign() >= 0 {
		return ErrInvalidAllocation
	}
	paid, err := money.NewMoney(strings.TrimPrefix(principal[0].Money.Amount(), "-"), principal[0].Money.Asset())
	if err != nil {
		return ErrInvalidAllocation
	}
	zero, _ := money.NewMoney("0", paid.Asset())
	total := zero
	seen := map[string]bool{}
	for _, item := range r.ReceiptItems {
		if seen[item.ID] || item.Validate() != nil || item.Gross.Asset() != paid.Asset() {
			return ErrInvalidAllocation
		}
		seen[item.ID] = true
		net, _ := item.Net()
		total, err = total.Add(net)
		if err != nil {
			return ErrInvalidAllocation
		}
	}
	if equal, _ := total.Compare(paid); equal != 0 {
		return ErrInvalidAllocation
	}
	return nil
}
