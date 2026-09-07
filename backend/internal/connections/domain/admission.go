package domain

import (
	"errors"
	"regexp"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
)

var (
	ErrInvalidAdmission    = errors.New("invalid deployment admission")
	ErrProviderNotAdmitted = errors.New("provider deployment not admitted")
	digestSyntax           = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	revisionSyntax         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
)

type Binding struct {
	Provider                   string
	Environment                string
	AdapterBuildDigest         string
	CollectorImageDigest       string
	ContractVersion            string
	AllowlistRevision          string
	NonSecretConfigRevision    string
	OperatorPermissionRevision string
}

func (b Binding) Validate() error {
	switch b.Provider {
	case "alfa", "raiffeisen", "ozon", "bybit", "aifory", "emcd":
	default:
		return ErrInvalidAdmission
	}
	if !digestSyntax.MatchString(b.AdapterBuildDigest) || !digestSyntax.MatchString(b.CollectorImageDigest) {
		return ErrInvalidAdmission
	}
	for _, value := range []string{b.Environment, b.ContractVersion, b.AllowlistRevision, b.NonSecretConfigRevision, b.OperatorPermissionRevision} {
		if !revisionSyntax.MatchString(value) {
			return ErrInvalidAdmission
		}
	}
	return nil
}

type AdmissionStatus string

const (
	Pending  AdmissionStatus = "pending"
	Admitted AdmissionStatus = "admitted"
	Blocked  AdmissionStatus = "blocked"
)

type CheckKind string

const (
	ProviderCheck CheckKind = "provider"
	HostCheck     CheckKind = "host"
)

type CheckResult string

const (
	CheckPassed  CheckResult = "passed"
	CheckFailed  CheckResult = "failed"
	CheckRevoked CheckResult = "revoked"
)

// Check is trusted application evidence. It is never constructed from a user or provider response DTO.
type Check struct {
	Kind    CheckKind
	Binding Binding
	Result  CheckResult
	At      calendar.Instant
}

type Admission struct {
	binding  Binding
	provider Check
	host     Check
	valid    bool
}

func NewAdmission(binding Binding) (Admission, error) {
	if err := binding.Validate(); err != nil {
		return Admission{}, err
	}
	return Admission{binding: binding, valid: true}, nil
}

func (a Admission) Binding() Binding { return a.binding }

func (a Admission) Status() AdmissionStatus {
	if a.provider.Result == CheckFailed || a.provider.Result == CheckRevoked || a.host.Result == CheckFailed || a.host.Result == CheckRevoked {
		return Blocked
	}
	if a.valid && a.provider.Result == CheckPassed && a.host.Result == CheckPassed {
		return Admitted
	}
	return Pending
}

func (a Admission) Reasons() []string {
	reasons := []string{}
	for _, check := range []struct {
		kind   string
		result CheckResult
	}{{"provider", a.provider.Result}, {"host", a.host.Result}} {
		if check.result != CheckPassed {
			result := string(check.result)
			if result == "" {
				result = "pending"
			}
			reasons = append(reasons, check.kind+"_"+result)
		}
	}
	return reasons
}

func (a Admission) CheckedAt() calendar.Instant {
	if a.provider.At.Time().After(a.host.At.Time()) {
		return a.provider.At
	}
	return a.host.At
}

func (a Admission) RecordCheck(check Check) (Admission, error) {
	if !a.valid || check.Binding != a.binding || check.At.String() == "" {
		return Admission{}, ErrInvalidAdmission
	}
	if check.Result != CheckPassed && check.Result != CheckFailed && check.Result != CheckRevoked {
		return Admission{}, ErrInvalidAdmission
	}
	var previous Check
	switch check.Kind {
	case ProviderCheck:
		previous = a.provider
	case HostCheck:
		previous = a.host
	default:
		return Admission{}, ErrInvalidAdmission
	}
	if previous.At.String() != "" && !check.At.Time().After(previous.At.Time()) {
		if check == previous {
			return a, nil
		}
		return Admission{}, ErrInvalidAdmission
	}
	if check.Kind == ProviderCheck {
		a.provider = check
	} else {
		a.host = check
	}
	return a, nil
}

func (a Admission) Rebind(binding Binding) (Admission, error) {
	if a.valid && binding == a.binding {
		return a, nil
	}
	return NewAdmission(binding)
}

// RequireSync is the domain precondition; the application must enforce it atomically before job creation.
func (a Admission) RequireSync(current Binding) error {
	if !a.valid || a.Status() != Admitted || current != a.binding {
		return ErrProviderNotAdmitted
	}
	return nil
}
