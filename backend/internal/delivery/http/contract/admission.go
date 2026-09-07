package contract

import (
	connection "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
)

// AdmissionToDTO exposes server state only; there is intentionally no command-side inverse converter.
func (b *Boundary) AdmissionToDTO(admission connection.Admission) (generated.DeploymentGate, error) {
	var out generated.DeploymentGate
	binding := admission.Binding()
	var mapped *generated.DeploymentBinding
	if err := binding.Validate(); err == nil {
		mapped = &generated.DeploymentBinding{Provider: generated.Provider(binding.Provider), Environment: binding.Environment, AdapterBuildDigest: binding.AdapterBuildDigest, CollectorImageDigest: binding.CollectorImageDigest, ContractVersion: binding.ContractVersion, AllowlistRevision: binding.AllowlistRevision, NonSecretConfigRevision: binding.NonSecretConfigRevision, OperatorPermissionRevision: binding.OperatorPermissionRevision}
	}
	checkedAt := admission.CheckedAt().String()
	var err error
	switch admission.Status() {
	case connection.Pending:
		reasons := []generated.PendingDeploymentGateReasons{}
		for _, reason := range admission.Reasons() {
			reasons = append(reasons, generated.PendingDeploymentGateReasons(reason))
		}
		dto := generated.PendingDeploymentGate{Status: "pending", Binding: mapped, Reasons: reasons}
		if checkedAt != "" {
			dto.CheckedAt = &checkedAt
		}
		err = out.FromPendingDeploymentGate(dto)
	case connection.Admitted:
		if mapped == nil {
			return out, ErrInvalidRequest
		}
		err = out.FromAdmittedDeploymentGate(generated.AdmittedDeploymentGate{Status: "admitted", Binding: *mapped, CheckedAt: checkedAt, Reasons: []generated.AdmittedDeploymentGateReasons{}})
	case connection.Blocked:
		if mapped == nil {
			return out, ErrInvalidRequest
		}
		reasons := []generated.BlockedDeploymentGateReasons{}
		for _, reason := range admission.Reasons() {
			reasons = append(reasons, generated.BlockedDeploymentGateReasons(reason))
		}
		err = out.FromBlockedDeploymentGate(generated.BlockedDeploymentGate{Status: "blocked", Binding: *mapped, CheckedAt: checkedAt, Reasons: reasons})
	default:
		return out, ErrInvalidRequest
	}
	if err != nil {
		return out, err
	}
	return out, b.validateDTO("DeploymentGate", out)
}
