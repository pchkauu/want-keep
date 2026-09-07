package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
)

func main() {
	path := flag.String("output", "", "New private keyring file")
	purpose := flag.String("purpose", "", "attachments or connections")
	flag.Parse()
	if *path == "" || flag.NArg() != 0 || cryptobox.Generate(*path, *purpose) != nil {
		fmt.Fprintln(os.Stderr, "Keyring was not created. Check purpose, private output directory and file availability.")
		os.Exit(1)
	}
}
