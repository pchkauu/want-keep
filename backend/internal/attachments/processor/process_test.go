package processor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessDeadlineAndCrashNeverReturnAcceptance(t *testing.T) {
	for _, body := range []string{"sleep 10", "kill -KILL $$", "printf invalid"} {
		t.Run(body, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "parser")
			if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			process := Process{executable: path}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			started := time.Now()
			result, err := process.Inspect(ctx, "application/pdf", []byte("synthetic"))
			if err == nil || len(result.Pages) != 0 || time.Since(started) > time.Second {
				t.Fatal("crash or timeout accepted or parser child survived", err)
			}
		})
	}
}
