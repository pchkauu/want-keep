package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pchkauu/want-keep/backend/internal/attachments/processor"
)

func main() {
	if run() != nil {
		fmt.Fprintln(os.Stderr, "Document processor unavailable; check its isolated runtime.")
		os.Exit(1)
	}
}
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	engine, err := processor.NewEngine("/usr/local/bin/qpdf", "/usr/local/bin/pdftoppm", os.TempDir())
	if err != nil {
		return err
	}
	if len(os.Args) == 2 && os.Args[1] == "inspect" {
		return engine.InspectStream(ctx, os.Stdin, os.Stdout)
	}
	if len(os.Args) != 1 {
		return errors.New("invalid processor mode")
	}
	runner, err := processor.NewProcess()
	if err != nil {
		return err
	}
	defer runner.Close()
	socket := "/run/want-keep/documents.sock"
	if info, err := os.Lstat(socket); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return errors.New("invalid processor socket")
		}
		connection, dialErr := net.DialTimeout("unix", socket, time.Second)
		if dialErr == nil {
			_ = connection.Close()
			return errors.New("processor already running")
		}
		if err = os.Remove(socket); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if !filepath.IsAbs(socket) {
		return errors.New("invalid processor socket")
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return err
	}
	defer listener.Close()
	if err = os.Chmod(socket, 0o600); err != nil {
		return err
	}
	server := &http.Server{Handler: processor.NewServer(runner), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 35 * time.Second, MaxHeaderBytes: 4096, BaseContext: func(net.Listener) context.Context { return ctx }}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
