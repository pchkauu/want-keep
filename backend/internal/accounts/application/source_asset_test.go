package application

import (
	"testing"

	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestConfirmedSourceAssetMappings(t *testing.T) {
	for _, tc := range []struct {
		provider, code, asset string
		valid                 bool
	}{
		{"raiffeisen", "RUR", "RUB", true}, {"ozon", "RUR", "RUB", true}, {"raiffeisen", "RUB", "RUB", true},
		{"bybit", "RUR", "RUB", false}, {"bybit", "USDC.E", "USDC", false}, {"ozon", "USD", "RUB", false},
	} {
		input := ImportInput{Provider: tc.provider, ExternalAssetCode: tc.code}
		input.Asset = money.Asset(tc.asset)
		if (input.ValidateAsset() == nil) != tc.valid {
			t.Fatal(tc)
		}
	}
}
