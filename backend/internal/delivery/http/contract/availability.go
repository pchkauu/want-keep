package contract

import (
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func (b *Boundary) AmountFromDTO(value generated.AmountValue) (reporting.Amount, error) {
	if err := b.validateDTO("AmountValue", value); err != nil {
		return reporting.Amount{}, err
	}
	known, err := value.AsKnownAmount()
	if err != nil {
		return reporting.Amount{}, ErrInvalidRequest
	}
	if known.Knowledge == "known" {
		amount, err := (MoneyConverter{}).FromDTO(known.Value)
		if err != nil {
			return reporting.Amount{}, err
		}
		return reporting.KnownAmount(amount)
	}
	missing, err := value.AsMissingAmount()
	if err != nil {
		return reporting.Amount{}, ErrInvalidRequest
	}
	return reporting.MissingAmount(reporting.Knowledge(missing.Knowledge), missing.Reason)
}

func (b *Boundary) AmountToDTO(value reporting.Amount) (generated.AmountValue, error) {
	if err := value.Validate(); err != nil {
		return generated.AmountValue{}, err
	}
	var result generated.AmountValue
	if amount, ok := value.Value(); ok {
		dto, err := (MoneyConverter{}).ToDTO(amount)
		if err != nil {
			return result, err
		}
		if err := result.FromKnownAmount(generated.KnownAmount{Knowledge: "known", Value: dto}); err != nil {
			return result, err
		}
	} else {
		if err := result.FromMissingAmount(generated.MissingAmount{Knowledge: generated.MissingAmountKnowledge(value.Knowledge()), Reason: value.Reason()}); err != nil {
			return result, err
		}
	}
	return result, b.validateDTO("AmountValue", result)
}

func (b *Boundary) CoverageFromDTO(value generated.Coverage) (reporting.Coverage, error) {
	if err := b.validateDTO("Coverage", value); err != nil {
		return reporting.Coverage{}, err
	}
	complete, err := value.AsCompleteCoverage()
	if err != nil {
		return reporting.Coverage{}, ErrInvalidRequest
	}
	return reporting.NewCoverage(reporting.CoverageState(complete.State), complete.Reasons)
}

func (b *Boundary) CoverageToDTO(value reporting.Coverage) (generated.Coverage, error) {
	checked, err := reporting.NewCoverage(value.State(), value.Reasons())
	if err != nil {
		return generated.Coverage{}, err
	}
	var result generated.Coverage
	if checked.State() == reporting.Complete {
		err = result.FromCompleteCoverage(generated.CompleteCoverage{State: "complete", Reasons: []string{}})
	} else {
		err = result.FromIncompleteCoverage(generated.IncompleteCoverage{State: generated.IncompleteCoverageState(checked.State()), Reasons: checked.Reasons()})
	}
	if err != nil {
		return result, err
	}
	return result, b.validateDTO("Coverage", result)
}
