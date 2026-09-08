package application

import (
	"context"
	"errors"

	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
)

var ErrGatewayFailure = errors.New("AI gateway failure")

type GatewayFailure struct {
	Code              string
	Retryable         bool
	ConfirmedNoCharge bool
	OutcomeUnknown    bool
	Observation       ProviderObservation
}

func (e GatewayFailure) Error() string { return ErrGatewayFailure.Error() }
func (e GatewayFailure) Unwrap() error { return ErrGatewayFailure }

type ProviderObservation struct {
	ID, Model string
	Usage     *ai.Usage
}

type Gateway interface {
	Contract() ai.RuntimeContract
	Count(context.Context, ai.Request) (int64, error)
	Generate(context.Context, ai.Request) (ai.Result, error)
}
