package domain

import "errors"

type Asset string

const (
	RUB  Asset = "RUB"
	USD  Asset = "USD"
	USDT Asset = "USDT"
	USDC Asset = "USDC"
	BTC  Asset = "BTC"
	ETH  Asset = "ETH"
)

var ErrUnsupportedAsset = errors.New("unsupported asset")

func ParseAsset(code string) (Asset, error) {
	switch Asset(code) {
	case RUB, USD, USDT, USDC, BTC, ETH:
		return Asset(code), nil
	default:
		return "", ErrUnsupportedAsset
	}
}
