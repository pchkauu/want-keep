package domain

type Snapshot struct {
	Binding        Binding
	Revision       int64
	Provider, Host Check
}

func (a Admission) Snapshot() Snapshot { return Snapshot{a.binding, a.revision, a.provider, a.host} }

// Restore validates saved evidence without replaying transitions or resetting the counter.
func Restore(s Snapshot) (Admission, error) {
	if s.Revision < 1 || s.Revision > 9007199254740991 {
		return Admission{}, ErrInvalidAdmission
	}
	a, err := NewAdmission(s.Binding)
	if err != nil {
		return Admission{}, err
	}
	for _, entry := range []struct {
		check Check
		kind  CheckKind
	}{{s.Provider, ProviderCheck}, {s.Host, HostCheck}} {
		if entry.check == (Check{}) {
			continue
		}
		if entry.check.Kind != entry.kind || entry.check.Binding != s.Binding || entry.check.At.String() == "" {
			return Admission{}, ErrInvalidAdmission
		}
		if entry.check.Result != CheckPassed && entry.check.Result != CheckFailed && entry.check.Result != CheckRevoked {
			return Admission{}, ErrInvalidAdmission
		}
	}
	minimum := int64(1)
	if s.Provider != (Check{}) {
		minimum++
	}
	if s.Host != (Check{}) {
		minimum++
	}
	if s.Revision < minimum {
		return Admission{}, ErrInvalidAdmission
	}
	a.provider, a.host, a.revision = s.Provider, s.Host, s.Revision
	return a, nil
}
