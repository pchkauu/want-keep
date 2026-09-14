package contract

import (
	"context"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
)

type RawGateway interface {
	CapabilityManifest(context.Context) ([]byte, error)
	Read(context.Context, []byte) ([]byte, error)
}

// Gateway is the only adapter that sees the generated wire model. Provider-specific
// implementations exchange bytes and cannot leak their DTOs into application or domain code.
type Gateway struct {
	binding connections.Binding
	raw     RawGateway
}

func NewGateway(binding connections.Binding, raw RawGateway) (*Gateway, error) {
	if raw == nil || binding.Validate() != nil || binding.ContractVersion != ingestion.ContractVersion {
		return nil, ingestion.ErrInvalidContract
	}
	return &Gateway{binding: binding, raw: raw}, nil
}

func (g *Gateway) Binding() connections.Binding { return g.binding }

func (g *Gateway) Manifest(ctx context.Context) (ingestion.Manifest, error) {
	payload, err := g.raw.CapabilityManifest(ctx)
	if err != nil {
		return ingestion.Manifest{}, err
	}
	return DecodeManifest(payload, g.binding.Provider)
}

func (g *Gateway) Read(ctx context.Context, token ingestion.JobToken) (ingestion.Result, error) {
	if token.Binding != g.binding {
		return ingestion.Result{}, ingestion.ErrInvalidContract
	}
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
