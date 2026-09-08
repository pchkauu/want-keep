package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
)

type Config struct {
	APIKeyFile  string
	ProjectID   string
	BaseURL     string
	Environment string
	HTTPClient  *http.Client
}

type Client struct {
	sdk      openaisdk.Client
	contract runtimeContract
}

func New(config Config) (*Client, error) {
	if err := validateEndpoint(config.Environment, config.BaseURL); err != nil {
		return nil, err
	}
	contract, err := loadRuntimeContract()
	if err != nil {
		return nil, err
	}
	if config.Environment == "production" && !contract.ProductionAdmitted {
		return nil, errors.New("OpenAI runtime contract is not admitted for production")
	}
	key, err := readAPIKey(config.APIKeyFile)
	if err != nil {
		return nil, err
	}
	options := []option.RequestOption{option.WithAPIKey(key), option.WithMaxRetries(0)}
	if config.ProjectID != "" {
		options = append(options, option.WithProject(config.ProjectID))
	}
	if config.BaseURL != "" {
		options = append(options, option.WithBaseURL(config.BaseURL))
	}
	if config.HTTPClient != nil {
		options = append(options, option.WithHTTPClient(config.HTTPClient))
	}
	return &Client{sdk: openaisdk.NewClient(options...), contract: contract}, nil
}

func validateEndpoint(environment, baseURL string) error {
	if environment != "production" && environment != "development" && environment != "test" {
		return errors.New("invalid OpenAI environment")
	}
	if baseURL == "" {
		return nil
	}
	if environment == "production" {
		return errors.New("custom OpenAI endpoint is forbidden in production")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1" || parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("test OpenAI endpoint must be loopback")
	}
	return nil
}

func readAPIKey(path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) {
		return "", errors.New("OpenAI API key file is required")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() < 1 || info.Size() > 16*1024 {
		return "", errors.New("invalid OpenAI API key file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", errors.New("cannot read OpenAI API key file")
	}
	key := strings.TrimSpace(string(data))
	if key == "" || strings.ContainsAny(key, "\r\n\t ") {
		return "", errors.New("invalid OpenAI API key")
	}
	return key, nil
}

func (c *Client) Contract() ai.RuntimeContract { return c.contract.metadata() }

func (c *Client) Count(ctx context.Context, request ai.Request) (int64, error) {
	if err := c.validateRequest(request); err != nil {
		return 0, err
	}
	response, err := c.sdk.Responses.InputTokens.Count(ctx, responses.InputTokenCountParams{
		Instructions:      openaisdk.String(c.contract.Prompt),
		Model:             openaisdk.String(c.contract.Model),
		ParallelToolCalls: openaisdk.Bool(false),
		Input: responses.InputTokenCountParamsInputUnion{
			OfString: openaisdk.String(string(request.Input)),
		},
		Reasoning: shared.ReasoningParam{Effort: shared.ReasoningEffortXhigh, Mode: shared.ReasoningModeStandard},
		Text:      responses.InputTokenCountParamsText{Format: c.textFormat()},
	})
	if err != nil {
		return 0, classifyError(err, false)
	}
	maximumInput, _ := request.Purpose.Limits()
	if !response.JSON.InputTokens.Valid() || response.InputTokens <= 0 || response.InputTokens > maximumInput || response.InputTokens > c.contract.GlobalMaximumInput {
		return 0, aiapp.GatewayFailure{Code: "input_limit", Retryable: false}
	}
	return response.InputTokens, nil
}

func (c *Client) Generate(ctx context.Context, request ai.Request) (ai.Result, error) {
	if err := c.validateRequest(request); err != nil {
		return ai.Result{}, err
	}
	expected, err := proposalExpectation(request.Input)
	if err != nil {
		return ai.Result{}, ai.ErrInvalidAttempt
	}
	response, err := c.sdk.Responses.New(ctx, responses.ResponseNewParams{
		Background:         openaisdk.Bool(false),
		Input:              responses.ResponseNewParamsInputUnion{OfString: openaisdk.String(string(request.Input))},
		Instructions:       openaisdk.String(c.contract.Prompt),
		MaxOutputTokens:    openaisdk.Int(request.MaximumOutputTokens),
		Model:              shared.ResponsesModel(c.contract.Model),
		ParallelToolCalls:  openaisdk.Bool(false),
		PromptCacheOptions: responses.ResponseNewParamsPromptCacheOptions{Mode: "explicit"},
		Reasoning:          shared.ReasoningParam{Effort: shared.ReasoningEffortXhigh, Mode: shared.ReasoningModeStandard},
		ServiceTier:        responses.ResponseNewParamsServiceTierDefault,
		Store:              openaisdk.Bool(false),
		Text:               responses.ResponseTextConfigParam{Format: c.textFormat()},
		Truncation:         responses.ResponseNewParamsTruncationDisabled,
	})
	if err != nil {
		return ai.Result{}, classifyError(err, true)
	}
	observation, usageOK := providerObservation(response)
	if !usageOK {
		return ai.Result{}, aiapp.GatewayFailure{Code: "usage_invalid", OutcomeUnknown: true, Observation: observation}
	}
	if observation.Model != string(request.Model) {
		return ai.Result{}, aiapp.GatewayFailure{Code: "model_mismatch", OutcomeUnknown: true, Observation: observation}
	}
	result := ai.Result{
		ProviderID: observation.ID, ProviderModel: observation.Model,
		State: ai.Completed, Output: json.RawMessage(response.OutputText()), Usage: *observation.Usage,
	}
	switch response.Status {
	case responses.ResponseStatusCompleted:
		if hasRefusal(response.RawJSON()) {
			result.State, result.Code, result.Output = ai.Refused, "model_refusal", nil
		} else if err := validateProposal(result.Output, expected); err != nil {
			result.State, result.Code, result.Output = ai.SchemaError, "invalid_schema", nil
		}
	case responses.ResponseStatusIncomplete:
		result.State, result.Code, result.Output = ai.Incomplete, "incomplete", nil
	default:
		result.State, result.Code, result.Output = ai.SchemaError, "unexpected_status", nil
	}
	if err := result.Validate(); err != nil {
		return ai.Result{}, aiapp.GatewayFailure{Code: "invalid_response", OutcomeUnknown: true, Observation: observation}
	}
	return result, nil
}

func providerObservation(response *responses.Response) (aiapp.ProviderObservation, bool) {
	observation := aiapp.ProviderObservation{}
	if response.JSON.ID.Valid() {
		observation.ID = response.ID
	}
	if response.JSON.Model.Valid() {
		observation.Model = string(response.Model)
	}
	usage := response.Usage
	valid := response.JSON.ID.Valid() && response.ID != "" && response.JSON.Model.Valid() && observation.Model != "" &&
		response.JSON.Status.Valid() && response.JSON.Usage.Valid() && usage.JSON.InputTokens.Valid() &&
		usage.JSON.InputTokensDetails.Valid() && usage.InputTokensDetails.JSON.CachedTokens.Valid() &&
		usage.JSON.OutputTokens.Valid() && usage.JSON.OutputTokensDetails.Valid() &&
		usage.OutputTokensDetails.JSON.ReasoningTokens.Valid() && usage.JSON.TotalTokens.Valid()
	if !valid {
		return observation, false
	}
	value := ai.Usage{
		InputTokens: usage.InputTokens, CachedTokens: usage.InputTokensDetails.CachedTokens,
		OutputTokens: usage.OutputTokens, ReasoningTokens: usage.OutputTokensDetails.ReasoningTokens,
	}
	if usage.InputTokensDetails.JSON.CacheWriteTokens.Valid() {
		writes := usage.InputTokensDetails.CacheWriteTokens
		value.CacheWriteTokens = &writes
	}
	if value.Validate() != nil || usage.TotalTokens != usage.InputTokens+usage.OutputTokens {
		return observation, false
	}
	observation.Usage = &value
	return observation, true
}

func (c *Client) textFormat() responses.ResponseFormatTextConfigUnionParam {
	return responses.ResponseFormatTextConfigUnionParam{OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
		Name: "want_keep_proposal", Schema: c.contract.Schema, Strict: openaisdk.Bool(true),
	}}
}

func (c *Client) validateRequest(request ai.Request) error {
	if err := request.Validate(); err != nil {
		return err
	}
	if _, err := proposalExpectation(request.Input); err != nil {
		return ai.ErrInvalidAttempt
	}
	contract := c.Contract()
	if request.Model != contract.Model || request.Qualification != contract.Qualification || request.PromptFingerprint != contract.PromptFingerprint || request.SchemaFingerprint != contract.SchemaFingerprint || request.ConfigFingerprint != contract.ConfigFingerprint {
		return ai.ErrInvalidAttempt
	}
	return nil
}

func classifyError(err error, generation bool) error {
	var apiError *openaisdk.Error
	if errors.As(err, &apiError) {
		status := apiError.StatusCode
		return aiapp.GatewayFailure{
			Code:      "provider_rejected",
			Retryable: status == http.StatusTooManyRequests || status >= 500,
		}
	}
	if generation && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || errors.Is(err, io.ErrUnexpectedEOF)) {
		return aiapp.GatewayFailure{Code: "provider_unknown", OutcomeUnknown: true}
	}
	return aiapp.GatewayFailure{Code: "provider_unavailable", Retryable: !generation, OutcomeUnknown: generation}
}
