package security

import (
	"context"
	"crypto/hmac"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	application "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
)

const SessionCookie = "want_keep_session"

type Config struct {
	Environment, Origin string
	TrustedProxies      []netip.Prefix
}
type Guard struct {
	config Config
	host   string
}
type Sessions interface {
	Me(context.Context, identity.Token) (application.Access, error)
}

func New(c Config) (*Guard, error) {
	u, err := url.Parse(c.Origin)
	if err != nil || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, identity.ErrAttempt
	}
	switch c.Environment {
	case "production":
		if c.Origin != "https://want-keep.tech" {
			return nil, identity.ErrAttempt
		}
	case "test", "development":
		if u.Hostname() != "localhost" || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, identity.ErrAttempt
		}
	default:
		return nil, identity.ErrAttempt
	}
	return &Guard{config: c, host: u.Host}, nil
}
func (g *Guard) Check(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	if r.Host != g.host {
		return identity.ErrAttempt
	}
	if g.config.Environment == "production" && r.TLS == nil && !(g.trustedPeer(r) && r.Header.Get("X-Forwarded-Proto") == "https") {
		return identity.ErrUnauthorized
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if r.Header.Get("Origin") != g.config.Origin || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			return identity.ErrUnauthorized
		}
	}
	return nil
}
func (g *Guard) trustedPeer(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	for _, prefix := range g.config.TrustedProxies {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}
func (g *Guard) Source(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", identity.ErrAttempt
	}
	if g.trustedPeer(r) {
		forwarded := r.Header.Get("X-Forwarded-For")
		if forwarded != "" {
			if strings.Contains(forwarded, ",") {
				return "", identity.ErrAttempt
			}
			if _, err = netip.ParseAddr(forwarded); err != nil {
				return "", identity.ErrAttempt
			}
			host = forwarded
		}
	}
	return host, nil
}
func (g *Guard) Authorize(r *http.Request, s Sessions, mutation bool) (application.Access, error) {
	cookie, err := r.Cookie(SessionCookie)
	if err != nil {
		return application.Access{}, identity.ErrUnauthorized
	}
	token := identity.Token(cookie.Value)
	if mutation && (!token.Valid(32) || !hmac.Equal([]byte(token.CSRF()), []byte(r.Header.Get("X-CSRF-Token")))) {
		return application.Access{}, identity.ErrUnauthorized
	}
	return s.Me(r.Context(), token)
}
