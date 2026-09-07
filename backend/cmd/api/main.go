package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	delivery "github.com/pchkauu/want-keep/backend/internal/delivery/identity"
	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "API stopped: check the private configuration and database availability.")
		os.Exit(1)
	}
}
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	environment, origin := os.Getenv("WANT_KEEP_ENV"), os.Getenv("WANT_KEEP_ORIGIN")
	u, err := url.Parse(origin)
	if err != nil {
		return err
	}
	verifier, err := webauthn.New(u.Hostname(), origin)
	if err != nil {
		return err
	}
	database, err := storage.Open(ctx, storage.Config{DSN: os.Getenv("WANT_KEEP_DATABASE_URL"), Environment: environment, MaxConnections: 8})
	if err != nil {
		return err
	}
	defer database.Close()
	maximum := 2
	if raw := os.Getenv("WANT_KEEP_MAX_MEMBERS"); raw != "" {
		maximum, err = strconv.Atoi(raw)
		if err != nil || maximum < 1 {
			return errors.New("invalid member limit")
		}
	}
	service, err := application.NewService(database, database, verifier, u.Hostname(), origin, maximum, time.Now)
	if err != nil {
		return err
	}
	config := delivery.Config{Environment: environment, Origin: origin}
	if raw := os.Getenv("WANT_KEEP_TRUSTED_PROXIES"); raw != "" {
		for _, value := range strings.Split(raw, ",") {
			prefix, err := netip.ParsePrefix(value)
			if err != nil {
				return err
			}
			config.TrustedProxies = append(config.TrustedProxies, prefix)
		}
	}
	handler, err := delivery.New(service, config)
	if err != nil {
		return err
	}
	address := os.Getenv("WANT_KEEP_LISTEN_ADDR")
	if address == "" {
		address = "127.0.0.1:8080"
	}
	server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	stopped := make(chan error, 1)
	go func() { stopped <- server.ListenAndServe() }()
	select {
	case err := <-stopped:
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
