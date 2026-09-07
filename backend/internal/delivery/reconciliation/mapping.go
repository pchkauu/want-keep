package reconciliation

import (
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	reconciliation "github.com/pchkauu/want-keep/backend/internal/reconciliation/domain"
)

func (s *Server) toDTO(value reconciliation.Reconciliation) (generated.Reconciliation, error) {
	out := generated.Reconciliation{
		Id: value.ID, AccountId: value.AccountID, Revision: int64(value.Revision),
		Lifecycle: generated.ReconciliationLifecycle(value.Lifecycle), Result: generated.ReconciliationResult(value.Result),
		SourceAsOf: value.SourceAsOf.String(), EvaluatedAt: value.EvaluatedAt.String(),
		Components: []generated.ReconciliationComponent{}, Explanations: []generated.ReconciliationExplanation{},
		RelatedTransactionIds: append([]string{}, value.RelatedOperationIDs...),
		Replay:                generated.ReconciliationReplay{Status: generated.ReconciliationReplayStatus(value.Replay.Status)},
	}
	coverage, err := s.boundary.CoverageToDTO(value.Coverage)
	if err != nil {
		return out, err
	}
	out.Quality.Coverage, out.Quality.Freshness = coverage, generated.Freshness(value.Freshness)
	asOf := value.SourceAsOf.String()
	out.Quality.AsOf = &asOf
	for _, component := range value.Components {
		source, err := s.boundary.AmountToDTO(component.Source)
		if err != nil {
			return out, err
		}
		ledger, err := s.boundary.AmountToDTO(component.Ledger)
		if err != nil {
			return out, err
		}
		difference, err := s.boundary.AmountToDTO(component.Difference)
		if err != nil {
			return out, err
		}
		out.Components = append(out.Components, generated.ReconciliationComponent{Component: generated.ReconciliationComponentComponent(component.Name), Source: source, Ledger: ledger, Difference: difference, Adjustable: component.Name.Adjustable()})
	}
	for _, explanation := range value.Explanations {
		out.Explanations = append(out.Explanations, generated.ReconciliationExplanation{Code: explanation.Code, Message: explanation.Message})
	}
	if value.Replay.Status != reconciliation.ReplayNotRequired {
		from, to := value.Replay.From.String(), value.Replay.To.String()
		out.Replay.RequestedFrom, out.Replay.RequestedTo = &from, &to
		if value.Replay.JobID != "" {
			out.Replay.JobId = &value.Replay.JobID
		}
		if value.Replay.Reason != "" {
			out.Replay.Reason = &value.Replay.Reason
		}
	}
	if value.Resolution != nil {
		components := make([]generated.ReconciliationResolutionComponents, len(value.Resolution.Components))
		for index, component := range value.Resolution.Components {
			components[index] = generated.ReconciliationResolutionComponents(component)
		}
		out.Resolution = &generated.ReconciliationResolution{ActorId: value.Resolution.ActorID, Reason: value.Resolution.Reason, AdjustmentTransactionId: value.Resolution.AdjustmentTransactionID, Components: components, RecordedAt: value.Resolution.At.String()}
	}
	return out, nil
}
