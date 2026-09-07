package application

import (
	"context"
	"encoding/base64"
	"strings"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Service) BeginBootstrap(ctx context.Context, r RequestContext, token identity.Token, setup identity.Setup) (Enrollment, error) {
	var result Enrollment
	if strings.TrimSpace(setup.Name) == "" || len(setup.Name) > 2000 || strings.TrimSpace(setup.HouseholdName) == "" || len(setup.HouseholdName) > 2000 || (setup.Locale != "ru" && setup.Locale != "en") {
		return result, identity.ErrAttempt
	}
	if _, err := calendar.ParseTimezone(setup.Timezone); err != nil {
		return result, identity.ErrAttempt
	}
	if _, err := money.ParseAsset(setup.ReportingAsset); err != nil {
		return result, identity.ErrAttempt
	}
	if err := s.Limit(ctx, r); err != nil {
		return result, err
	}
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		b, err := s.repo.IdentityBootstrap(ctx)
		if err != nil {
			return err
		}
		if err = b.Require(token, s.now()); err != nil {
			return err
		}
		user, err := s.newID()
		if err != nil {
			return err
		}
		family, err := s.newID()
		if err != nil {
			return err
		}
		member, err := s.newID()
		if err != nil {
			return err
		}
		handle, err := identity.NewToken(32)
		if err != nil {
			return err
		}
		setup.UserID, setup.HouseholdID, setup.MembershipID = household.UserID(user), household.HouseholdID(family), household.MembershipID(member)
		setup.Handle, _ = base64.RawURLEncoding.DecodeString(string(handle))
		a, err := s.attempt(identity.Bootstrap, r)
		if err != nil {
			return err
		}
		a.Setup = &setup
		a.GrantID = b.TokenHash
		result = Enrollment{Attempt: a, Profile: s.setupProfile(setup)}
		return s.repo.SaveIdentityAttempt(ctx, a)
	})
	return result, err
}
func (s *Service) setupProfile(x identity.Setup) identity.Profile {
	return identity.Profile{UserID: x.UserID, HouseholdID: x.HouseholdID, MembershipID: x.MembershipID, Name: x.Name, Handle: x.Handle, Generation: 1, Locale: x.Locale, ReportingAsset: x.ReportingAsset}
}
func (s *Service) BeginEnrollment(ctx context.Context, r RequestContext, purpose identity.Purpose, authorization identity.Token) (Enrollment, error) {
	var result Enrollment
	if purpose != identity.AddPasskey && purpose != identity.Recovery {
		return result, identity.ErrAttempt
	}
	if err := s.Limit(ctx, r); err != nil {
		return result, err
	}
	err := s.repo.WithinIdentity(ctx, func(ctx context.Context) error {
		a, err := s.attempt(purpose, r)
		if err != nil {
			return err
		}
		var p identity.Profile
		if purpose == identity.AddPasskey {
			access, err := s.access(ctx, r.Session)
			if err != nil {
				return err
			}
			if err = access.Session.RequireFresh(s.now()); err != nil {
				return err
			}
			p = access.Profile
			a.SessionHash = r.Session.Hash()
		} else {
			if !authorization.Valid(32) {
				return identity.ErrAttempt
			}
			grant, err := s.repo.IdentityGrant(ctx, authorization.Hash())
			if err != nil {
				return err
			}
			if err = grant.Require(r.Browser.Hash(), s.now(), identity.RecoveryGrant); err != nil {
				return err
			}
			p, err = s.repo.IdentityProfile(ctx, grant.UserID)
			if err != nil {
				return err
			}
			if p.Generation != grant.Generation {
				return identity.ErrAttempt
			}
			a.GrantID = grant.ID
		}
		if _, err = s.repo.IdentityMembership(ctx, p); err != nil {
			return err
		}
		if err = s.repo.IdentityRateLimit(ctx, "user:"+string(p.UserID), 30, s.now()); err != nil {
			return err
		}
		credentials, err := s.repo.IdentityCredentials(ctx, p.UserID)
		if err != nil {
			return err
		}
		active := 0
		for _, c := range credentials {
			if !c.Revoked {
				active++
				result.ExcludeIDs = append(result.ExcludeIDs, c.RawID)
			}
		}
		if purpose == identity.AddPasskey && active >= 20 {
			return identity.ErrAttempt
		}
		a.UserID, a.Generation = p.UserID, p.Generation
		result.Attempt, result.Profile = a, p
		return s.repo.SaveIdentityAttempt(ctx, a)
	})
	return result, err
}

func (s *Service) CompleteEnrollment(ctx context.Context, r RequestContext, id, name string, input Registration) (EnrollmentResult, error) {
	var result EnrollmentResult
	if strings.TrimSpace(name) == "" || len(name) > 2000 {
		return result, identity.ErrAttempt
	}
	if err := s.Limit(ctx, r); err != nil {
		return result, err
	}
	err := s.finish(ctx, r, id, func(ctx context.Context, a identity.Attempt) error {
		var p identity.Profile
		var err error
		switch a.Purpose {
		case identity.Bootstrap:
			b, err := s.repo.IdentityBootstrap(ctx)
			if err != nil {
				return err
			}
			if a.Setup == nil || b.Initialized || b.TokenHash != a.GrantID || !s.now().Before(b.ExpiresAt) {
				return identity.ErrBootstrap
			}
			p = s.setupProfile(*a.Setup)
		case identity.AddPasskey, identity.Recovery:
			p, err = s.repo.IdentityProfile(ctx, a.UserID)
			if err != nil {
				return err
			}
			if p.Generation != a.Generation {
				return identity.ErrAttempt
			}
			if _, err = s.repo.IdentityMembership(ctx, p); err != nil {
				return err
			}
			if a.Purpose == identity.AddPasskey {
				access, err := s.access(ctx, r.Session)
				if err != nil {
					return err
				}
				if access.Profile.UserID != p.UserID || a.SessionHash != r.Session.Hash() {
					return identity.ErrUnauthorized
				}
				if err = access.Session.RequireFresh(s.now()); err != nil {
					return err
				}
				keys, err := s.repo.IdentityCredentials(ctx, p.UserID)
				if err != nil {
					return err
				}
				active := 0
				for _, key := range keys {
					if !key.Revoked {
						active++
					}
				}
				if active >= 20 {
					return identity.ErrAttempt
				}
			} else {
				grant, err := s.repo.IdentityAttempt(ctx, a.GrantID)
				if err != nil {
					return err
				}
				if err = grant.Require(r.Browser.Hash(), s.now(), identity.RecoveryGrant); err != nil {
					return err
				}
				if grant.UserID != p.UserID || grant.Generation != p.Generation {
					return identity.ErrAttempt
				}
				grant.Consumed = true
				if err = s.repo.SaveIdentityAttempt(ctx, grant); err != nil {
					return err
				}
			}
		default:
			return identity.ErrAttempt
		}
		c, err := s.verifier.Register(p, a, input)
		if err != nil {
			return err
		}
		if _, err = s.repo.IdentityCredential(ctx, c.RawID); err == nil {
			return identity.ErrAttempt
		} else if err != identity.ErrUnauthorized {
			return err
		}
		c.ID, err = s.newID()
		if err != nil {
			return err
		}
		c.UserID, c.Name, c.CreatedAt = p.UserID, name, s.now()
		if a.Purpose == identity.Bootstrap {
			zone, _ := calendar.ParseTimezone(a.Setup.Timezone)
			if err = s.households.InitializeHousehold(ctx, household.Household{ID: p.HouseholdID, Name: a.Setup.HouseholdName}, []household.User{{ID: p.UserID, Name: p.Name}}, []household.Membership{{ID: p.MembershipID, UserID: p.UserID, HouseholdID: p.HouseholdID, Active: true}}, zone, s.maximum); err != nil {
				return err
			}
			if err = s.repo.CreateIdentityProfile(ctx, p); err != nil {
				return err
			}
			if err = s.repo.SaveIdentityBootstrap(ctx, identity.BootstrapState{Initialized: true}); err != nil {
				return err
			}
		}
		if a.Purpose == identity.Recovery {
			if err = s.revokeAll(ctx, p); err != nil {
				return err
			}
			p, err = p.Recovered()
			if err != nil {
				return err
			}
			if err = s.repo.SaveIdentityGeneration(ctx, p); err != nil {
				return err
			}
		}
		if err = s.repo.SaveIdentityCredential(ctx, c); err != nil {
			return err
		}
		result.Purpose = a.Purpose
		if a.Purpose == identity.AddPasskey {
			result.Access, err = s.access(ctx, r.Session)
		} else {
			result.Codes, err = s.replaceCodes(ctx, p)
			if err != nil {
				return err
			}
			result.Access, err = s.createSession(ctx, p, c.ID)
		}
		if err != nil {
			return err
		}
		event := "passkey_added"
		if a.Purpose == identity.Bootstrap {
			event = "bootstrap_completed"
		}
		if a.Purpose == identity.Recovery {
			event = "recovery_completed"
		}
		return s.repo.IdentityAudit(ctx, p.UserID, event, c.ID, s.now())
	})
	return result, err
}
