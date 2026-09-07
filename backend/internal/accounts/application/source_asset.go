package application

import money "github.com/pchkauu/want-keep/backend/internal/money/domain"

// ValidateAsset applies only mappings established by the provider contracts; the raw code is retained unchanged.
func (input ImportInput) ValidateAsset() error {
	asset, err := money.ParseAsset(input.ExternalAssetCode)
	if err != nil && input.ExternalAssetCode == "RUR" && (input.Provider == "raiffeisen" || input.Provider == "ozon") {
		asset, err = money.RUB, nil
	}
	if err != nil {
		return err
	}
	if asset != input.Asset {
		return money.ErrUnsupportedAsset
	}
	return nil
}
