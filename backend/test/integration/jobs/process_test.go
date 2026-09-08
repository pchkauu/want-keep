//go:build integration

package storage_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func TestCrashProcessHelper(t *testing.T) {
	mode := os.Getenv("WANT_KEEP_JOB_CRASH_MODE")
	if mode == "" {
		return
	}
	db, err := storage.Open(testContext, storage.Config{DSN: os.Getenv("WANT_KEEP_JOB_CRASH_DSN"), Environment: "test", MaxConnections: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.ClaimJobs(testContext, "outbox", 1, time.Minute)
	if err != nil || len(rows) != 1 {
		t.Fatal("child claim failed", err)
	}
	j := rows[0]
	p, err := db.JobPrincipal(testContext, j)
	if err != nil {
		t.Fatal(err)
	}
	block := func() {
		fmt.Println("READY")
		for {
			time.Sleep(time.Second)
		}
	}
	if mode == "external" {
		if err = db.BeginExternal(testContext, p, j); err != nil {
			t.Fatal(err)
		}
		block()
	}
	err = db.WithinHousehold(testContext, p, func(ctx context.Context) error {
		if err := db.EmitEvent(ctx, "synthetic", uuid.NewString(), 1, "synthetic.child.effect"); err != nil {
			return err
		}
		if mode == "before_commit" {
			block()
		}
		return db.FinishJob(ctx, p, j)
	})
	if err != nil {
		t.Fatal(err)
	}
	block()
}
func TestKilledProcessRecovery(t *testing.T) {
	for _, mode := range []string{"before_commit", "after_commit", "external"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			f.event()
			cmd := exec.Command(os.Args[0], "-test.run=^TestCrashProcessHelper$")
			cmd.Env = append(os.Environ(), "WANT_KEEP_JOB_CRASH_MODE="+mode, "WANT_KEEP_JOB_CRASH_DSN="+f.dsn)
			output, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill() })
			ready := make(chan bool, 1)
			go func() {
				scanner := bufio.NewScanner(output)
				for scanner.Scan() {
					if scanner.Text() == "READY" {
						ready <- true
						return
					}
				}
				ready <- false
			}()
			select {
			case ok := <-ready:
				if !ok {
					t.Fatal("child did not reach crash boundary")
				}
			case <-time.After(10 * time.Second):
				t.Fatal("child boundary timeout")
			}
			if err = cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = cmd.Wait()
			var n int
			if err = f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.outbox WHERE event_type='synthetic.child.effect'`).Scan(&n); err != nil {
				t.Fatal(err)
			}
			if (mode == "after_commit" && n != 1) || (mode != "after_commit" && n != 0) {
				t.Fatal("partial or lost effect", n)
			}
			if _, err = f.admin.Exec(testContext, `UPDATE want_keep.jobs SET lease_until=clock_timestamp()-INTERVAL '1 second' WHERE state='running'`); err != nil {
				t.Fatal(err)
			}
			rows, err := f.restart().ClaimJobs(testContext, "outbox", 10, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "external" && len(rows) != 0 {
				t.Fatal("external crash repeated")
			}
			if mode == "before_commit" && len(rows) != 1 {
				t.Fatal("rolled-back work lost")
			}
			if mode == "after_commit" && (len(rows) != 1 || f.count("job_receipts") != 1) {
				t.Fatal("completed job repeated or followup lost")
			}
		})
	}
}
