//go:build integration

package storage_test

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func binding() connections.Binding {
	return connections.Binding{Provider: "raiffeisen", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("a", 64), CollectorImageDigest: "sha256:" + strings.Repeat("b", 64), ContractVersion: "10", AllowlistRevision: "1", NonSecretConfigRevision: "1", OperatorPermissionRevision: "1"}
}
func (f *fixture) connection() string {
	f.t.Helper()
	id := uuid.NewString()
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.store.CreateConnection(ctx, admission.Connection{HouseholdID: f.family.ID, ID: id, Provider: "raiffeisen", Owner: f.p.UserID(), Generation: 1, Authorized: true})
	}); err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f *fixture) admit(b connections.Binding) *admission.Service {
	f.t.Helper()
	service := admission.NewService(f.store, f.store)
	for _, kind := range []connections.CheckKind{connections.ProviderCheck, connections.HostCheck} {
		if _, err := service.RecordCheck(testContext, connections.Check{Kind: kind, Binding: b, Result: connections.CheckPassed, At: f.now}); err != nil {
			f.t.Fatal(err)
		}
	}
	return service
}
func (f *fixture) issued(service *admission.Service, id string, b connections.Binding) jobs.Job {
	f.t.Helper()
	j, err := service.RequestSync(testContext, f.p, id, b, time.Now().Add(time.Hour))
	if err != nil {
		f.t.Fatal(err)
	}
	entries, err := f.store.ClaimJobs(testContext, "sync", 100, time.Minute)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.ID == j.ID {
			return entry
		}
	}
	f.t.Fatal("sync not claimable")
	return jobs.Job{}
}
