package smtp

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// handlerFunc handles one SMTP verb. It returns quit=true when the
// connection should be closed after the reply is sent (QUIT, or an
// unrecoverable protocol error such as a failed STARTTLS handshake).
type handlerFunc func(sess *session, arg string) (quit bool)

// handlers maps each supported verb to its handler. Dispatch is a map
// lookup (see session.dispatch), not an if/else or switch chain — adding a
// new verb later is a one-line entry here, not a growing conditional.
var handlers = map[string]handlerFunc{
	"HELO":     handleHELO,
	"EHLO":     handleEHLO,
	"MAIL":     handleMAILFROM,
	"RCPT":     handleRCPTTO,
	"DATA":     handleDATA,
	"RSET":     handleRSET,
	"NOOP":     handleNOOP,
	"QUIT":     handleQUIT,
	"VRFY":     handleVRFY,
	"STARTTLS": handleSTARTTLS,
	"AUTH":     handleAUTH,
}

// dispatch looks up and invokes the handler for verb, replying with
// codeCommandUnrecognized when no such verb is known.
func (sess *session) dispatch(verb, arg string) (quit bool) {
	h, ok := handlers[verb]
	if !ok {
		sess.reply(codeCommandUnrecognized, msgCommandUnrecognized)
		return false
	}
	return h(sess, arg)
}

func handleHELO(sess *session, arg string) bool {
	sess.heloDomain = arg
	sess.state = stateHelo
	sess.resetTransaction()
	sess.reply(codeOK, fmt.Sprintf(msgHeloFmt, sess.srv.cfg.Domain, arg))
	return false
}

func handleEHLO(sess *session, arg string) bool {
	sess.heloDomain = arg
	sess.state = stateHelo
	sess.resetTransaction()

	lines := append([]string{fmt.Sprintf(msgHeloFmt, sess.srv.cfg.Domain, arg)}, ehloCapabilities(sess)...)
	sess.replyMultiline(codeOK, lines)
	return false
}

// ehloCapabilities builds the EHLO capability list from a small table rather
// than nested conditionals — each capability's presence is a single boolean
// derived from config/session state.
func ehloCapabilities(sess *session) []string {
	caps := []struct {
		text    string
		enabled bool
	}{
		{"8BITMIME", true},
		{fmt.Sprintf("SIZE %d", sess.srv.cfg.MaxMessageSize), true},
		{"STARTTLS", sess.srv.cfg.StartTLS && !sess.tlsActive},
		{"AUTH PLAIN LOGIN", sess.srv.cfg.AuthEnabled},
	}
	var out []string
	for _, c := range caps {
		if c.enabled {
			out = append(out, c.text)
		}
	}
	return out
}

func handleMAILFROM(sess *session, arg string) bool {
	if sess.state == stateGreeted {
		sess.reply(codeBadSequence, msgHeloRequiredFirst)
		return false
	}
	if sess.srv.cfg.AuthEnabled && !sess.authed {
		sess.reply(codeAuthRequired, msgAuthRequired)
		return false
	}

	addr, params, ok := parseMailParams(arg, "FROM:")
	if !ok {
		sess.reply(codeSyntaxError, msgSyntaxError)
		return false
	}
	if declared, ok := params["SIZE"]; ok {
		if n, err := strconv.ParseInt(declared, 10, 64); err == nil && n > sess.srv.cfg.MaxMessageSize {
			sess.reply(codeExceededStorage, msgSizeExceeded)
			return false
		}
	}

	sess.envFrom = addr
	sess.envTo = nil
	sess.state = stateMail
	sess.reply(codeOK, msgMailOK)
	return false
}

func handleRCPTTO(sess *session, arg string) bool {
	if sess.state != stateMail && sess.state != stateRcpt {
		sess.reply(codeBadSequence, msgMailRequiredFirst)
		return false
	}

	addr, _, ok := parseMailParams(arg, "TO:")
	if !ok {
		sess.reply(codeSyntaxError, msgSyntaxError)
		return false
	}

	// maelsink is a sink, not a real MTA: RCPT TO is never rejected for an
	// unknown/nonexistent recipient (SPEC.md §4).
	sess.envTo = append(sess.envTo, addr)
	sess.state = stateRcpt
	sess.reply(codeOK, msgRcptOK)
	return false
}

func handleDATA(sess *session, _ string) bool {
	if sess.state != stateRcpt {
		sess.reply(codeBadSequence, msgRcptRequiredFirst)
		return false
	}

	sess.reply(codeStartMailInput, msgStartMailInput)
	start := time.Now()

	dr := sess.tp.DotReader()
	limited := io.LimitReader(dr, sess.srv.cfg.MaxMessageSize+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		sess.reply(codeLocalError, msgLocalError)
		return true
	}
	if int64(len(raw)) > sess.srv.cfg.MaxMessageSize {
		// Simplification for a local dev tool: rather than draining and
		// resyncing an oversized dot-terminated payload, close the
		// connection after rejecting it.
		sess.reply(codeExceededStorage, msgSizeExceeded)
		return true
	}

	msg := Parse(raw)
	msg.ReceivedAt = time.Now()
	msg.EnvelopeFrom = sess.envFrom
	msg.EnvelopeTo = sess.envTo
	msg.Bcc = deriveBcc(msg.To, msg.Cc, sess.envTo)

	ctx := context.Background()
	if err := sess.srv.store.Save(ctx, msg); err != nil {
		sess.srv.logger.Error("smtp: failed to save message", "err", err)
		sess.reply(codeLocalError, msgLocalError)
		sess.state = stateHelo
		sess.resetTransaction()
		return false
	}
	sess.srv.publisher.Publish(ctx, msg)

	sess.srv.logger.Info("smtp: message accepted",
		"msg_id", msg.ID,
		"from", msg.EnvelopeFrom,
		"to", strings.Join(msg.EnvelopeTo, ","),
		"size", msg.Size,
		"attachments", len(msg.Attachments),
		"parse_warning", msg.ParseWarning,
		"duration", time.Since(start),
	)

	sess.reply(codeOK, fmt.Sprintf(msgQueuedFmt, msg.ID))
	sess.state = stateHelo
	sess.resetTransaction()
	return false
}

func handleRSET(sess *session, _ string) bool {
	sess.resetTransaction()
	if sess.heloDomain != "" {
		sess.state = stateHelo
	} else {
		sess.state = stateGreeted
	}
	sess.reply(codeOK, msgOK)
	return false
}

func handleNOOP(sess *session, _ string) bool {
	sess.reply(codeOK, msgOK)
	return false
}

func handleQUIT(sess *session, _ string) bool {
	sess.reply(codeClosing, msgBye)
	return true
}

func handleVRFY(sess *session, _ string) bool {
	// Never confirms or denies whether a specific mailbox exists.
	sess.reply(codeCannotVRFY, msgVRFYNotSupported)
	return false
}

func handleSTARTTLS(sess *session, _ string) bool {
	if !sess.srv.cfg.StartTLS {
		sess.reply(codeCommandNotImplemented, msgTLSNotOffered)
		return false
	}
	if sess.tlsActive {
		sess.reply(codeBadSequence, msgTLSAlreadyActive)
		return false
	}

	sess.reply(codeGreeting, msgReadyStartTLS)
	if err := sess.startTLS(); err != nil {
		sess.srv.logger.Warn("smtp: starttls handshake failed", "err", err)
		return true
	}
	return false
}

// parseMailParams parses the argument of a MAIL FROM/RCPT TO command, e.g.
// "FROM:<user@example.com> SIZE=1024" with prefix "FROM:", into the bare
// address and any trailing "KEY=VALUE" parameters.
func parseMailParams(arg, prefix string) (addr string, params map[string]string, ok bool) {
	if !strings.HasPrefix(strings.ToUpper(arg), prefix) {
		return "", nil, false
	}
	rest := strings.TrimSpace(arg[len(prefix):])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "", nil, false
	}

	addr = strings.Trim(fields[0], "<>")
	params = make(map[string]string, len(fields)-1)
	for _, f := range fields[1:] {
		kv := strings.SplitN(f, "=", 2)
		if len(kv) == 2 {
			params[strings.ToUpper(kv[0])] = kv[1]
		}
	}
	return addr, params, true
}
