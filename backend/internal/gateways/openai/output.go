package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
)

var (
	amountPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)
	monthPattern  = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])$`)
)

type proposalEnvelope struct {
	Results []proposal `json:"results"`
}

type proposal struct {
	ID          string   `json:"id"`
	Action      string   `json:"action"`
	Kind        *string  `json:"kind"`
	Amount      *string  `json:"amount"`
	Fee         *string  `json:"fee"`
	Asset       *string  `json:"asset"`
	Target      *string  `json:"target"`
	Month       *string  `json:"month"`
	Shares      []share  `json:"shares"`
	Items       []string `json:"items"`
	Evidence    []string `json:"evidence"`
	Explanation string   `json:"explanation"`
}

type share struct {
	Member string `json:"member"`
	Amount string `json:"amount"`
}

func validateProposal(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var envelope proposalEnvelope
	if err := decoder.Decode(&envelope); err != nil || decoder.Decode(new(any)) != io.EOF || envelope.Results == nil {
		return errors.New("invalid proposal")
	}
	for _, result := range envelope.Results {
		if result.ID == "" || result.Explanation == "" || !oneOf(result.Action, "create", "link", "clarify", "reject", "skip", "explain") || result.Shares == nil || result.Items == nil || result.Evidence == nil {
			return errors.New("invalid proposal")
		}
		if (result.Kind != nil && !oneOf(*result.Kind, "expense", "income", "refund", "transfer", "exchange")) ||
			(result.Asset != nil && !oneOf(*result.Asset, "RUB", "USD", "USDT", "BTC", "ETH")) ||
			(result.Amount != nil && !amountPattern.MatchString(*result.Amount)) ||
			(result.Fee != nil && !amountPattern.MatchString(*result.Fee)) ||
			(result.Month != nil && !monthPattern.MatchString(*result.Month)) {
			return errors.New("invalid proposal")
		}
		for _, item := range result.Items {
			if !amountPattern.MatchString(item) {
				return errors.New("invalid proposal")
			}
		}
		for _, item := range result.Shares {
			if item.Member == "" || !amountPattern.MatchString(item.Amount) {
				return errors.New("invalid proposal")
			}
		}
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func hasRefusal(raw string) bool {
	var response struct {
		Output []struct {
			Content []struct {
				Type string `json:"type"`
			} `json:"content"`
		} `json:"output"`
	}
	if json.Unmarshal([]byte(raw), &response) != nil {
		return false
	}
	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Type == "refusal" {
				return true
			}
		}
	}
	return false
}
