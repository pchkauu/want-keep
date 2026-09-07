package domain_test

import (
	"errors"
	"math"
	"testing"
	"time"

	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

func TestAccessBoundariesAndImmutableTransitions(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	session := identity.Session{ID: "session", CreatedAt: now, AuthenticatedAt: now, LastActivityAt: now}
	if session.RequireActive(now.Add(30*time.Minute-time.Nanosecond)) != nil || session.RequireActive(now.Add(30*time.Minute)) == nil {
		t.Fatal("idle boundary")
	}
	if session.RequireFresh(now.Add(5*time.Minute-time.Nanosecond)) != nil || !errors.Is(session.RequireFresh(now.Add(5*time.Minute)), identity.ErrFreshAuthentication) {
		t.Fatal("fresh boundary")
	}
	next, err := session.Activity(now.Add(time.Minute))
	if err != nil || !session.LastActivityAt.Equal(now) || !next.LastActivityAt.Equal(now.Add(time.Minute)) {
		t.Fatal("activity mutates original")
	}
	profile := identity.Profile{Generation: math.MaxInt64}
	if _, err = profile.Recovered(); err == nil || profile.Generation != math.MaxInt64 {
		t.Fatal("generation overflow")
	}
	token, err := identity.NewToken(32)
	if err != nil || !token.Valid(32) || token.Valid(16) || len(token.Hash()) != 64 || token.CSRF() == token.Hash() {
		t.Fatal("token shape")
	}
	a := identity.Attempt{ID: "attempt", BrowserHash: token.Hash(), Purpose: identity.Login, CreatedAt: now, ExpiresAt: now.Add(5 * time.Minute)}
	if a.Require(token.Hash(), now, identity.Login) != nil || a.Require(token.Hash(), a.ExpiresAt, identity.Login) == nil || a.Require("other", now, identity.Login) == nil || a.Require(token.Hash(), now, identity.Recovery) == nil {
		t.Fatal("attempt binding")
	}
}
