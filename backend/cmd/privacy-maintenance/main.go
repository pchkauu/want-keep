package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pchkauu/want-keep/backend/internal/attachments/files"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	keys, err := cryptobox.Load(os.Getenv("WANT_KEEP_ATTACHMENT_KEYRING"), "attachments")
	if err == nil {
		var store *files.Store
		store, err = files.Open(os.Getenv("WANT_KEEP_ATTACHMENT_DIRECTORY"), keys)
		if err == nil {
			defer store.Close()
			_, err = store.CleanupPending(ctx, time.Now().UTC())
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Private object maintenance failed.")
		os.Exit(1)
	}
}
