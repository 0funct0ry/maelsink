package smtp

import (
	"crypto/subtle"
	"encoding/base64"
	"strings"
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
	return fn(sess, initial)
}

// handleAuthPlain implements RFC 4616 AUTH PLAIN: a single base64 blob of
// "authzid\x00username\x00password".
func handleAuthPlain(sess *session, initial string) bool {
	blob := initial
	if blob == "" {
		sess.reply(codeAuthContinue, "")
		line, err := sess.tp.ReadLine()
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
		line, err := sess.tp.ReadLine()
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

// credentialsMatch compares against the single configured username/password
// in constant time, avoiding a timing side channel on credential checks.
func credentialsMatch(sess *session, username, password string) bool {
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(sess.srv.cfg.AuthUsername)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(sess.srv.cfg.AuthPassword)) == 1
	return userOK && passOK
}
