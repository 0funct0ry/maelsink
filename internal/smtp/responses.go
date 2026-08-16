package smtp

// Response codes, per RFC 5321 (base SMTP), RFC 3207 (STARTTLS), and
// RFC 4954 (AUTH). Each constant documents the RFC status class and the
// situation in which maelsink sends it.
const (
	// codeGreeting is the initial connect banner and the STARTTLS-ready reply.
	codeGreeting = 220
	// codeClosing replies to QUIT: service closing transmission channel.
	codeClosing = 221
	// codeAuthSuccess (RFC 4954) confirms a completed AUTH exchange.
	codeAuthSuccess = 235
	// codeOK is the generic "requested action completed" reply used by
	// HELO, EHLO, MAIL, RCPT, RSET, NOOP, and successful DATA.
	codeOK = 250
	// codeCannotVRFY replies to VRFY: never confirms or denies a specific
	// mailbox, but always indicates the server will accept mail for it.
	codeCannotVRFY = 252
	// codeAuthContinue is the server challenge continuation used mid-AUTH
	// (e.g. the base64 "Username:"/"Password:" prompts of AUTH LOGIN).
	codeAuthContinue = 334
	// codeStartMailInput replies to DATA, inviting the client to send the
	// message terminated by "<CRLF>.<CRLF>".
	codeStartMailInput = 354
	// codeCommandUnrecognized: syntax error, command unrecognized (unknown verb).
	codeCommandUnrecognized = 500
	// codeSyntaxError: syntax error in command parameters or arguments.
	codeSyntaxError = 501
	// codeCommandNotImplemented: the command is recognized but not enabled
	// by configuration (e.g. STARTTLS or AUTH when disabled).
	codeCommandNotImplemented = 502
	// codeBadSequence: the command is valid but out of order for the
	// current transaction state (e.g. RCPT before MAIL, DATA before RCPT).
	codeBadSequence = 503
	// codeAuthRequired: smtp.auth.enabled is true and the client has not
	// yet authenticated.
	codeAuthRequired = 530
	// codeAuthFailed: the presented AUTH credentials did not match.
	codeAuthFailed = 535
	// codeEncryptionRequired (RFC 4954): a plaintext AUTH mechanism (PLAIN,
	// LOGIN) was attempted over an unprotected connection.
	codeEncryptionRequired = 538
	// codeExceededStorage: the message (or its declared SIZE) exceeds
	// smtp.max_message_size_mb.
	codeExceededStorage = 552
	// codeLocalError: the message was accepted for transmission but a
	// local error (e.g. a storage failure) prevented processing it.
	codeLocalError = 451
)

// Response message text, paired with the codes above. Kept as named
// constants so wording changes happen in exactly one place.
const (
	msgGreetingFmt           = "%s ESMTP maelsink" // formatted with cfg.Domain
	msgHeloFmt               = "%s Hello %s"       // formatted with cfg.Domain, client-supplied domain
	msgBye                   = "2.0.0 Bye"
	msgAuthSuccessful        = "2.7.0 Authentication successful"
	msgOK                    = "2.0.0 OK"
	msgMailOK                = "2.1.0 OK"
	msgRcptOK                = "2.1.5 OK"
	msgStartMailInput        = "Start mail input; end with <CRLF>.<CRLF>"
	msgQueuedFmt             = "2.0.0 OK: queued as %s" // formatted with the message ID
	msgVRFYNotSupported      = "2.1.5 Cannot VRFY user, but will accept message"
	msgAuthUsernamePrompt    = "Username:" // base64-encoded before being sent
	msgAuthPasswordPrompt    = "Password:"
	msgReadyStartTLS         = "2.0.0 Ready to start TLS"
	msgCommandUnrecognized   = "5.5.2 Command not recognized"
	msgSyntaxError           = "5.5.4 Syntax error in parameters or arguments"
	msgBadSequence           = "5.5.1 Bad sequence of commands"
	msgHeloRequiredFirst     = "5.5.1 HELO/EHLO required first"
	msgMailRequiredFirst     = "5.5.1 MAIL FROM required first"
	msgRcptRequiredFirst     = "5.5.1 RCPT TO required first"
	msgCommandNotImplemented = "5.5.1 Command not implemented"
	msgAuthRequired          = "5.7.0 Authentication required"
	msgAuthAlreadyDone       = "5.5.1 Already authenticated"
	msgAuthInvalid           = "5.7.8 Authentication credentials invalid"
	msgAuthCanceled          = "5.7.0 Authentication canceled"
	msgAuthUnsupportedMech   = "5.5.4 Unrecognized authentication mechanism"
	msgSizeExceeded          = "5.3.4 Message size exceeds fixed limit"
	msgTLSAlreadyActive      = "5.5.1 TLS already active"
	msgTLSNotOffered         = "5.5.1 STARTTLS not offered"
	msgTLSNotAvailable       = "5.5.1 STARTTLS not available"
	msgTLSHandshakeFailed    = "5.7.0 TLS handshake failed"
	msgMustStartTLS          = "5.7.0 Must issue a STARTTLS command first"
	msgEncryptionRequired    = "5.7.11 Encryption required for requested authentication mechanism"
	msgLocalError            = "4.3.0 Requested action aborted: local error in processing"
)
