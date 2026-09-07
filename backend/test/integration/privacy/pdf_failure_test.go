//go:build integration

package privacy_test

import (
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	app "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"github.com/pchkauu/want-keep/backend/internal/attachments/processor"
)

func TestKilledPDFToolRemainsRetryable(t *testing.T) {
	f := newFixture(t)
	directory := t.TempDir()
	engine, err := processor.NewEngine("/out/privacy-faultprobe", "/usr/local/bin/pdftoppm", directory)
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(directory, "processor.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := http.Server{Handler: processor.NewServer(engine)}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	client, err := processor.NewClient(socket)
	if err != nil {
		t.Fatal(err)
	}
	service := app.NewService(f.store, f.blobs, client)
	id := uuid.NewString()
	requireStatus(t, f.upload(f.ta, id, f.accountID, "application/pdf", pdfDocument(1, "")), 200)
	if _, err := client.Inspect(testContext, "application/pdf", pdfDocument(1, "")); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatal("killed tool did not return HTTP unavailability", err)
	}
	if err := service.ProcessNext(testContext); err != nil {
		t.Fatal("failed to persist retry state", err)
	}
	a := f.metadata(f.ta, id)
	if a.State != domain.Uploaded || a.Reason != domain.WaitingProcessor {
		t.Fatal("tool crash permanently rejected upload", a.State, a.Reason)
	}
	requireStatus(t, f.request(f.tb, "GET", "/attachments/"+id+"/content", nil, ""), 409)
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.attachments SET available_at=clock_timestamp() WHERE household_id=$1 AND id=$2`, f.a.HouseholdID(), id); err != nil {
		t.Fatal(err)
	}
	if err := f.service.ProcessNext(testContext); err != nil {
		t.Fatal(err)
	}
	if f.metadata(f.ta, id).State != domain.Accepted {
		t.Fatal("same upload did not recover using healthy pinned processor")
	}
}
