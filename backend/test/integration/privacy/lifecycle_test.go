//go:build integration

package privacy_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	app "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"github.com/pchkauu/want-keep/backend/internal/attachments/files"
	"github.com/pchkauu/want-keep/backend/internal/attachments/processor"
	delivery "github.com/pchkauu/want-keep/backend/internal/delivery/attachments"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	identitydelivery "github.com/pchkauu/want-keep/backend/internal/delivery/identity"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func TestUploadReplayConcurrencyAndEncryptedRestart(t *testing.T) {
	f := newFixture(t)
	id := uuid.NewString()
	data := imageDocument(t, "image/png")
	start := make(chan struct{})
	results := make(chan int, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			u := domain.Upload{ID: id, AccountID: f.accountID, Name: "receipt", MediaType: "image/png", Size: int64(len(data)), Hash: domain.Hash(data)}
			_, err := f.service.Upload(testContext, f.a, u, data)
			if err == nil {
				results <- 200
			} else {
				results <- 503
			}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	for status := range results {
		if status != 200 {
			t.Fatalf("concurrent upload: %d", status)
		}
	}
	requireStatus(t, f.upload(f.ta, id, f.accountID, "image/png", append(append([]byte{}, data...), 0)), 409)
	requireStatus(t, f.upload(f.tb, id, f.accountID, "image/png", data), 409)
	if err := f.service.ProcessNext(testContext); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.upload(f.ta, id, f.accountID, "image/png", data), 200)
	entries, err := os.ReadDir(f.blobDir)
	if err != nil || len(entries) != 2 {
		t.Fatal("duplicate objects", err, len(entries))
	}
	for _, entry := range entries {
		cipher, err := os.ReadFile(filepath.Join(f.blobDir, entry.Name()))
		if err != nil || bytes.Contains(cipher, data) || bytes.HasPrefix(cipher, []byte("\x89PNG")) {
			t.Fatal("plaintext object", err)
		}
	}
	var count int
	if err = f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.attachments`).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate metadata", err)
	}
	restarted, err := storage.Open(testContext, storage.Config{DSN: f.dsn, Environment: "test", MaxConnections: 4})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	ring, err := cryptobox.Load(f.keyPath, "attachments")
	if err != nil {
		t.Fatal(err)
	}
	blobs, err := files.Open(f.blobDir, ring)
	if err != nil {
		t.Fatal(err)
	}
	defer blobs.Close()
	service := app.NewService(restarted, blobs, documentProcessor)
	_, restored, err := service.Content(testContext, f.b, id, 0)
	if err != nil || !bytes.Equal(restored, data) {
		t.Fatal("restart changed original", err)
	}
	// The API database role cannot erase metadata, audit or immutable page identities.
	for _, sql := range []string{`DELETE FROM want_keep.attachments`, `UPDATE want_keep.attachment_pages SET page=2`, `UPDATE want_keep.privacy_audit SET event='attachment_ready'`, `DELETE FROM want_keep.privacy_audit`} {
		if _, err = f.appExec(sql); err == nil {
			t.Fatal("private history writable", sql)
		}
	}
}
func (f *fixture) appExec(sql string) (int64, error) {
	pool, err := pgxpool.New(testContext, f.dsn)
	if err != nil {
		return 0, err
	}
	defer pool.Close()
	tag, err := pool.Exec(testContext, sql)
	return tag.RowsAffected(), err
}

type lostResultRepository struct {
	*storage.Store
	calls        int
	failAfter    int
	beforeCommit bool
}

func (r *lostResultRepository) WithinHousehold(ctx context.Context, p household.Principal, fn func(context.Context) error) error {
	r.calls++
	err := r.Store.WithinHousehold(ctx, p, func(ctx context.Context) error {
		if err := fn(ctx); err != nil {
			return err
		}
		if r.calls == r.failAfter && r.beforeCommit {
			return errors.New("synthetic rollback")
		}
		return nil
	})
	if err == nil && r.calls == r.failAfter {
		return errors.New("synthetic lost acknowledgement")
	}
	return err
}
func TestUploadUnknownCommitAndRollbackAreRecoverable(t *testing.T) {
	for _, stage := range []struct {
		name     string
		after    int
		rollback bool
	}{{"registration-ack", 1, false}, {"ready-rollback", 2, true}, {"ready-ack", 2, false}} {
		t.Run(stage.name, func(t *testing.T) {
			f := newFixture(t)
			data := imageDocument(t, "image/png")
			u := domain.Upload{ID: uuid.NewString(), AccountID: f.accountID, Name: "receipt", MediaType: "image/png", Size: int64(len(data)), Hash: domain.Hash(data)}
			repo := &lostResultRepository{Store: f.store, failAfter: stage.after, beforeCommit: stage.rollback}
			service := app.NewService(repo, f.blobs, documentProcessor)
			if _, err := service.Upload(testContext, f.a, u, data); err == nil {
				t.Fatal("fault not injected")
			}
			a := f.metadata(f.ta, u.ID)
			if a.State != domain.Uploaded {
				t.Fatal("unfinished accepted")
			}
			if _, _, err := service.Content(testContext, f.a, u.ID, 0); !errors.Is(err, domain.ErrNotReady) {
				t.Fatal("unfinished content exposed")
			}
			if _, err := f.service.Upload(testContext, f.a, u, data); err != nil {
				t.Fatal(err)
			}
			if err := f.service.ProcessNext(testContext); err != nil {
				t.Fatal(err)
			}
			if f.metadata(f.ta, u.ID).State != domain.Accepted {
				t.Fatal("recovery failed")
			}
		})
	}
}
func TestPendingProcessorAndLeaseRecovery(t *testing.T) {
	f := newFixture(t)
	data := imageDocument(t, "image/png")
	id := uuid.NewString()
	requireStatus(t, f.upload(f.ta, id, f.accountID, "image/png", data), 200)
	unavailable, _ := processor.NewClient(filepath.Join(t.TempDir(), "missing.sock"))
	waiting := app.NewService(f.store, f.blobs, unavailable)
	if err := waiting.ProcessNext(testContext); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatal("missing processor accepted")
	}
	if f.metadata(f.ta, id).State != domain.Uploaded {
		t.Fatal("missing processor consumed job")
	}
	attempt, ok, err := f.store.ClaimAttachment(testContext)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `UPDATE want_keep.attachments SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE household_id=$1 AND id=$2`, f.a.HouseholdID(), id); err != nil {
		t.Fatal(err)
	}
	restarted, ok, err := f.store.ClaimAttachment(testContext)
	if err != nil || !ok || restarted.Token == attempt.Token {
		t.Fatal("expired lease not recovered", err)
	}
	if err = f.store.DeferAttachment(testContext, attempt, domain.WaitingProcessor); !errors.Is(err, domain.ErrStale) {
		t.Fatal("stale worker updated state", err)
	}
	if err = f.store.DeferAttachment(testContext, restarted, domain.WaitingProcessor); err != nil {
		t.Fatal(err)
	}
	if _, err = f.admin.Exec(testContext, `UPDATE want_keep.attachments SET available_at=clock_timestamp() WHERE household_id=$1 AND id=$2`, f.a.HouseholdID(), id); err != nil {
		t.Fatal(err)
	}
	if err = f.service.ProcessNext(testContext); err != nil {
		t.Fatal(err)
	}
	if f.metadata(f.ta, id).State != domain.Accepted {
		t.Fatal("resumed validation failed")
	}
}
func TestPrivateOutageLeavesAuthenticationAvailable(t *testing.T) {
	f := newFixture(t)
	service := app.NewService(f.store, nil, documentProcessor)
	private, err := delivery.New(service, f.identity, security.Config{Environment: "test", Origin: "http://localhost"}, func() bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	auth, err := identitydelivery.New(f.identity, security.Config{Environment: "test", Origin: "http://localhost"})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/attachments", private)
	mux.Handle("/api/v1/system/privacy", private)
	mux.Handle("/", auth)
	f.handler = mux
	requireStatus(t, f.request(f.ta, "GET", "/me", nil, ""), 200)
	requireStatus(t, f.upload(f.ta, uuid.NewString(), f.accountID, "image/png", imageDocument(t, "image/png")), 503)
	status := f.request(f.tb, "GET", "/system/privacy", nil, "")
	requireStatus(t, status, 200)
	if bytes.Contains(status.Body.Bytes(), []byte(f.keyPath)) || !bytes.Contains(status.Body.Bytes(), []byte(`"attachments":"unavailable"`)) {
		t.Fatal("unsafe or incorrect health")
	}
	if err = f.store.WithinHousehold(testContext, f.a, func(ctx context.Context) error { _, err := f.store.Account(ctx, f.a, f.accountID); return err }); err != nil {
		t.Fatal("accounting stopped with private storage", err)
	}
}
func TestProcessorCancellationDoesNotAcceptAndRemainsAvailable(t *testing.T) {
	ctx, cancel := context.WithDeadline(testContext, time.Now().Add(-time.Second))
	defer cancel()
	if _, err := documentProcessor.Inspect(ctx, "application/pdf", pdfDocument(1, "")); err == nil {
		t.Fatal("canceled request accepted")
	}
	result, err := documentProcessor.Inspect(testContext, "application/pdf", pdfDocument(1, ""))
	if err != nil || len(result.Pages) != 1 {
		t.Fatal("processor did not recover", err, result.Reason)
	}
}

type filesystemFaultRepository struct {
	*storage.Store
	directory string
	injected  bool
}

func (r *filesystemFaultRepository) WithinHousehold(ctx context.Context, p household.Principal, fn func(context.Context) error) error {
	err := r.Store.WithinHousehold(ctx, p, fn)
	if err == nil && !r.injected {
		r.injected = true
		return os.Chmod(r.directory, 0o500)
	}
	return err
}
func TestRealFileWriteAndPublicationFailureRecover(t *testing.T) {
	for _, failure := range []string{"write", "publish"} {
		t.Run(failure, func(t *testing.T) {
			f := newFixture(t)
			data := imageDocument(t, "image/png")
			u := domain.Upload{ID: uuid.NewString(), AccountID: f.accountID, Name: "receipt", MediaType: "image/png", Size: int64(len(data)), Hash: domain.Hash(data)}
			var service *app.Service
			var blockedPath string
			if failure == "write" {
				service = app.NewService(&filesystemFaultRepository{Store: f.store, directory: f.blobDir}, f.blobs, documentProcessor)
			} else {
				var a domain.Attachment
				if err := f.store.WithinHousehold(testContext, f.a, func(ctx context.Context) error {
					var err error
					a, err = f.store.ReserveAttachment(ctx, f.a, u)
					return err
				}); err != nil {
					t.Fatal(err)
				}
				blockedPath = filepath.Join(f.blobDir, domain.Hash([]byte(string(a.HouseholdID)+"/"+a.ObjectID+"/original"))+".blob")
				if err := os.Mkdir(blockedPath, 0o700); err != nil {
					t.Fatal(err)
				}
				service = f.service
			}
			if _, err := service.Upload(testContext, f.a, u, data); err == nil {
				t.Fatal("filesystem failure accepted")
			}
			if err := os.Chmod(f.blobDir, 0o700); err != nil {
				t.Fatal(err)
			}
			if blockedPath != "" {
				if err := os.Remove(blockedPath); err != nil {
					t.Fatal(err)
				}
			}
			a := f.metadata(f.ta, u.ID)
			if a.OriginalReady || a.State != domain.Uploaded {
				t.Fatal("unfinished write published")
			}
			if _, err := f.service.Upload(testContext, f.a, u, data); err != nil {
				t.Fatal(err)
			}
			if err := f.service.ProcessNext(testContext); err != nil {
				t.Fatal(err)
			}
			if f.metadata(f.ta, u.ID).State != domain.Accepted {
				t.Fatal("file recovery failed")
			}
		})
	}
}
func TestValidationMetadataRollbackKeepsPreviewsPrivate(t *testing.T) {
	f := newFixture(t)
	id := uuid.NewString()
	requireStatus(t, f.upload(f.ta, id, f.accountID, "image/png", imageDocument(t, "image/png")), 200)
	_, err := f.admin.Exec(testContext, `CREATE FUNCTION want_keep.synthetic_accept_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.event='attachment_accepted' THEN RAISE EXCEPTION 'synthetic commit failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER synthetic_accept_failure BEFORE INSERT ON want_keep.privacy_audit FOR EACH ROW EXECUTE FUNCTION want_keep.synthetic_accept_failure()`)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.ProcessNext(testContext); err == nil {
		t.Fatal("rollback not injected")
	}
	requireStatus(t, f.request(f.tb, "GET", "/attachments/"+id+"/preview/1", nil, ""), 409)
	var pages int
	if err = f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.attachment_pages`).Scan(&pages); err != nil || pages != 0 {
		t.Fatal("partial metadata", err)
	}
	_, err = f.admin.Exec(testContext, `DROP TRIGGER synthetic_accept_failure ON want_keep.privacy_audit; UPDATE want_keep.attachments SET lease_until=clock_timestamp()-INTERVAL '1 second'`)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.ProcessNext(testContext); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.request(f.tb, "GET", "/attachments/"+id+"/preview/1", nil, ""), 200)
}
