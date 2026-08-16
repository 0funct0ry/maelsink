package smtp

import (
	"crypto/subtle"
	"encoding/base64"
	"strings"

	"github.com/0funct0ry/maelsink/internal/webauth"
)

// authMechanismFunc implements one SASL mechanism for AUTH. initial is the
// mechanism's initial-response argument, if the client supplied one inline
// (e.g. "AUTH PLAIN <base64>"); empty if the client expects a challenge.
// It returns quit=true only when the connection must be closed (a read
// failure mid-exchange).
type authMechanismFunc func(sess *session, initial string) (quit bool)

// authMechanisms maps each supported SASL mechanism name to its handler —
// a lookup table instead of an if/else on the mechanism name, mirroring the
// verb dispatch table in commands.go.
var authMechanisms = map[string]authMechanismFunc{
	"PLAIN": handleAuthPlain,
	"LOGIN": handleAuthLogin,
}

func handleAUTH(sess *session, arg string) bool {
	if !sess.srv.cfg.AuthEnabled {
		sess.reply(codeCommandNotImplemented, msgCommandNotImplemented)
		return false
	}
	if sess.authed {
		sess.reply(codeBadSequence, msgAuthAlreadyDone)
		return false
	}
	if sess.srv.cfg.RequireStartTLS && !sess.tlsActive {
		sess.reply(codeAuthRequired, msgMustStartTLS)
		return false
	}
	// RFC 4954's closing MUST: plaintext AUTH mechanisms (PLAIN, LOGIN —
	// the only two supported) are refused unless the session is already
	// protected by STARTTLS, implicit TLS (RequireTLS), or the operator has
	// explicitly opted into insecure local/CI auth.
	if !sess.tlsActive && !sess.srv.cfg.RequireTLS && !sess.srv.cfg.AuthAllowInsecure {
		sess.reply(codeEncryptionRequired, msgEncryptionRequired)
		return false
	}

	fields := strings.SplitN(arg, " ", 2)
	mechanism := strings.ToUpper(fields[0])
	var initial string
	if len(fields) > 1 {
		initial = fields[1]
	}

	fn, ok := authMechanisms[mechanism]
	if !ok {
		sess.reply(codeSyntaxError, msgAuthUnsupportedMech)
		return false
	}

	// Mark the exchange in progress so readLine redacts any continuation
	// line fn reads on our behalf (these bypass run()'s main loop
	// entirely), and always clear it once the exchange concludes —
	// success, failure, or a read error that ends the connection.
	sess.authMechanism = mechanism
	defer func() { sess.authMechanism = "" }()
	return fn(sess, initial)
}

// handleAuthPlain implements RFC 4616 AUTH PLAIN: a single base64 blob of
// "authzid\x00username\x00password".
func handleAuthPlain(sess *session, initial string) bool {
	blob := initial
	if blob == "" {
		sess.reply(codeAuthContinue, "")
		line, err := sess.readLine()
		if err != nil {
			return true
		}
		blob = line
	}

	decoded, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		sess.reply(codeAuthFailed, msgAuthInvalid)
		return false
	}

	parts := strings.Split(string(decoded), "\x00")
	if len(parts) != 3 {
		sess.reply(codeAuthFailed, msgAuthInvalid)
		return false
	}
	username, password := parts[1], parts[2]

	if !credentialsMatch(sess, username, password) {
		sess.reply(codeAuthFailed, msgAuthInvalid)
		return false
	}
	sess.authed = true
	sess.reply(codeAuthSuccess, msgAuthSuccessful)
	return false
}

// handleAuthLogin implements the (non-standard but universally supported)
// AUTH LOGIN mechanism: base64-encoded "Username:"/"Password:" prompts,
// each answered with a base64-encoded value on the next line.
func handleAuthLogin(sess *session, initial string) bool {
	username, ok := sess.readAuthPrompt(msgAuthUsernamePrompt, initial)
	if !ok {
		return true
	}
	password, ok := sess.readAuthPrompt(msgAuthPasswordPrompt, "")
	if !ok {
		return true
	}

	if !credentialsMatch(sess, username, password) {
		sess.reply(codeAuthFailed, msgAuthInvalid)
		return false
	}
	sess.authed = true
	sess.reply(codeAuthSuccess, msgAuthSuccessful)
	return false
}

// readAuthPrompt sends a base64-encoded challenge (unless initial already
// supplies the answer) and returns the base64-decoded client response.
func (sess *session) readAuthPrompt(prompt, initial string) (value string, ok bool) {
	blob := initial
	if blob == "" {
		sess.reply(codeAuthContinue, base64.StdEncoding.EncodeToString([]byte(prompt)))
		line, err := sess.readLine()
		if err != nil {
			return "", false
		}
		blob = line
	}

	decoded, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		return "", false
	}
	return string(decoded), true
}

// credentialsMatch resolves an AUTH PLAIN/LOGIN attempt against every
// configured credential source, in this order, short-circuiting on first
// match: (1) accept_any, a pure test/CI escape hatch; (2) the single
// configured username/password pair; (3) MAELSINK_SMTP_AUTH's in-memory
// map; (4) the smtp.auth.file htpasswd-style store. All comparisons stay
// constant-time to avoid timing-based username enumeration.
func credentialsMatch(sess *session, username, password string) bool {
	cfg := sess.srv.cfg

	if cfg.AuthAcceptAny {
		return true
	}

	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(cfg.AuthUsername)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(cfg.AuthPassword)) == 1
	if userOK && passOK {
		return true
	}

	if extraMatch(cfg.AuthExtraCredentials, username, password) {
		return true
	}

	if cfg.AuthFile != "" && webauth.Verify(cfg.AuthFile, username, password) {
		return true
	}

	return false
}

// extraMatch checks username/password against MAELSINK_SMTP_AUTH's parsed
// map in constant time: every entry is compared (not just a map lookup by
// username first) so the number of configured users, and which usernames
// exist, isn't leaked via response timing.
func extraMatch(creds map[string]string, username, password string) bool {
	matched := false
	for u, p := range creds {
		userOK := subtle.ConstantTimeCompare([]byte(username), []byte(u)) == 1
		passOK := subtle.ConstantTimeCompare([]byte(password), []byte(p)) == 1
		if userOK && passOK {
			matched = true
		}
	}
	return matched
}
