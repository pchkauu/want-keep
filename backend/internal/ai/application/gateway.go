package application

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
)

var ErrGatewayFailure = errors.New("AI gateway failure")

type GatewayFailure struct {
	Code              string
	Retryable         bool
	ConfirmedNoCharge bool
	OutcomeUnknown    bool
	RetryAfter        time.Duration
	Observation       ProviderObservation
}

func (e GatewayFailure) Error() string { return ErrGatewayFailure.Error() }
func (e GatewayFailure) Unwrap() error { return ErrGatewayFailure }

type ProviderObservation struct {
	ID, Model string
	Usage     *ObservedUsage
}

// ObservedUsage preserves the bounded numeric provider evidence separately from
// the validated usage used for accounting. Pointer fields retain field presence.
type ObservedUsage struct {
	InputTokens      *int64 `json:"input_tokens,omitempty"`
	CachedTokens     *int64 `json:"cached_tokens,omitempty"`
	CacheWriteTokens *int64 `json:"cache_write_tokens,omitempty"`
	OutputTokens     *int64 `json:"output_tokens,omitempty"`
	ReasoningTokens  *int64 `json:"reasoning_tokens,omitempty"`
	TotalTokens      *int64 `json:"total_tokens,omitempty"`
}

func ObserveUsage(usage ai.Usage) *ObservedUsage {
	input, cached, output, reasoning := usage.InputTokens, usage.CachedTokens, usage.OutputTokens, usage.ReasoningTokens
	total := input + output
	observed := &ObservedUsage{
		InputTokens: &input, CachedTokens: &cached,
		OutputTokens: &output, ReasoningTokens: &reasoning, TotalTokens: &total,
	}
	if usage.CacheWriteTokens != nil {
		value := *usage.CacheWriteTokens
		observed.CacheWriteTokens = &value
	}
	return observed
}

func (u ObservedUsage) Validate() error {
	if u.InputTokens == nil && u.CachedTokens == nil && u.CacheWriteTokens == nil && u.OutputTokens == nil && u.ReasoningTokens == nil && u.TotalTokens == nil {
		return ai.ErrInvalidUsage
	}
	encoded, err := json.Marshal(u)
	if err != nil || len(encoded) > 2048 {
		return ai.ErrInvalidUsage
	}
	return nil
}

func (u ObservedUsage) Exact() (ai.Usage, bool) {
	if u.InputTokens == nil || u.CachedTokens == nil || u.OutputTokens == nil || u.ReasoningTokens == nil {
		return ai.Usage{}, false
	}
	value := ai.Usage{
		InputTokens: *u.InputTokens, CachedTokens: *u.CachedTokens, CacheWriteTokens: u.CacheWriteTokens,
		OutputTokens: *u.OutputTokens, ReasoningTokens: *u.ReasoningTokens,
	}
	if value.Validate() != nil || value.InputTokens > math.MaxInt64-value.OutputTokens || u.TotalTokens != nil && *u.TotalTokens != value.InputTokens+value.OutputTokens {
		return ai.Usage{}, false
	}
	return value, true
}

type Gateway interface {
	Contract() ai.RuntimeContract
	Count(context.Context, ai.Request) (int64, error)
	Generate(context.Context, ai.Request) (ai.Result, error)
}
