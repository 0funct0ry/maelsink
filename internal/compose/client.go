package compose

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/0funct0ry/maelsink/internal/cliclient"
)

// TargetConfig describes the target maelsink instance compose connects to
// (SPEC.md §7.7.6) — the process it proxies for, not compose's own server.
type TargetConfig struct {
	APIAddr               string
	APIUser               string
	APIPass               string
	APIInsecureSkipVerify bool
	APICACert             string

	// SMTPAddr/SMTPUser/SMTPPass are accepted now so M13.1's Composer
	// (stateless render/send) needs no further config wiring, though M13.0's
	// handlers don't use them yet.
	SMTPAddr string
	SMTPUser string
	SMTPPass string
}

// basicAuthTransport injects HTTP Basic Auth for whatever fronts the
// target's REST API (e.g. the Web UI port with M8.8's htpasswd gate
// enabled) — distinct from the target API's own Bearer api_key auth, which
// cliclient.Client already supports natively via its APIKey field.
type basicAuthTransport struct {
	user, pass string
	inner      http.RoundTripper
}

func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.user != "" || t.pass != "" {
		req.SetBasicAuth(t.user, t.pass)
	}
	return t.inner.RoundTrip(req)
}

// NewTargetClient builds a cliclient.Client configured from cfg, injecting
// Basic Auth and TLS settings server-side so the browser never sees target
// credentials (SPEC.md §7.7.3).
func NewTargetClient(cfg TargetConfig) (*cliclient.Client, error) {
	client := cliclient.NewClient(cfg.APIAddr, "")

	tlsConfig := &tls.Config{InsecureSkipVerify: cfg.APIInsecureSkipVerify}
	if cfg.APICACert != "" {
		pem, err := os.ReadFile(cfg.APICACert)
		if err != nil {
			return nil, fmt.Errorf("reading api-ca-cert %q: %w", cfg.APICACert, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("api-ca-cert %q: no valid certificates found", cfg.APICACert)
		}
		tlsConfig.RootCAs = pool
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client.HTTPClient = &http.Client{
		Transport: &basicAuthTransport{user: cfg.APIUser, pass: cfg.APIPass, inner: transport},
	}

	return client, nil
}
