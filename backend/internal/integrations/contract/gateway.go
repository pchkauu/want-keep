package contract

import (
	"context"

	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
)

type RawGateway interface {
	CapabilityManifest(context.Context) ([]byte, error)
	Read(context.Context, []byte) ([]byte, error)
}

// Gateway is the only adapter that sees the generated wire model. Provider-specific
// implementations exchange bytes and cannot leak their DTOs into application or domain code.
type Gateway struct {
	provider string
	raw      RawGateway
}

func NewGateway(provider string, raw RawGateway) (*Gateway, error) {
	switch provider {
	case "alfa", "raiffeisen", "ozon", "bybit", "aifory", "emcd":
	default:
		return nil, ingestion.ErrInvalidContract
	}
	if raw == nil {
		return nil, ingestion.ErrInvalidContract
	}
	return &Gateway{provider: provider, raw: raw}, nil
}

func (g *Gateway) Manifest(ctx context.Context) (ingestion.Manifest, error) {
	payload, err := g.raw.CapabilityManifest(ctx)
	if err != nil {
		return ingestion.Manifest{}, err
	}
	return DecodeManifest(payload, g.provider)
}

func (g *Gateway) Read(ctx context.Context, token ingestion.JobToken) (ingestion.Result, error) {
	request, err := EncodeSyncRequest(token)
	if err != nil {
		return ingestion.Result{}, err
	}
	payload, err := g.raw.Read(ctx, request)
	if err != nil {
		return ingestion.Result{}, err
	}
	return DecodeResult(payload, token)
}
