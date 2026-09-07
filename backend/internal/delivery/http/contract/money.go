package contract

import (
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type MoneyConverter struct{}

func (MoneyConverter) FromDTO(value generated.Money) (money.Money, error) {
	asset, err := money.ParseAsset(string(value.Asset))
	if err != nil {
		return money.Money{}, err
	}
	return money.NewMoney(value.Amount, asset)
}

func (MoneyConverter) ToDTO(value money.Money) (generated.Money, error) {
	if err := value.Validate(); err != nil {
		return generated.Money{}, err
	}
	return generated.Money{Amount: value.Amount(), Asset: generated.Asset(value.Asset())}, nil
}

func (MoneyConverter) RateFromDTO(value generated.Rate) (money.Rate, error) {
	return money.NewRate(money.Asset(value.Base), money.Asset(value.Quote), value.Value)
}

func (MoneyConverter) RateToDTO(value money.Rate) (generated.Rate, error) {
	if err := value.Validate(); err != nil {
		return generated.Rate{}, err
	}
	return generated.Rate{Base: generated.Asset(value.Base()), Quote: generated.Asset(value.Quote()), Value: value.Value()}, nil
}
