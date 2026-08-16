package compose

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/mail"
	"net/textproto"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// renderRequest is the shared request body for /render and /send (SPEC.md
// §7.7.4.1): a template document plus the vars map to seed it with. Vars is
// a plain map[string]string, mirroring web-compose's localStorage-backed
// vars store (VarsMap) 1:1 — no server-side session or persistence, per
// SPEC.md §7.7.1/§7.7.3.
//
// Attachments only applies to Format=="eml": raw RFC 5322 text has no
// structured place to say "attach this file", unlike the "json" format's
// cliclient.MessageSpec, which carries its own Attachments field inline in
// Template. It is ignored for Format=="json".
type renderRequest struct {
	Template    string                     `json:"template"`
	Format      string                     `json:"format"`
	Vars        map[string]string          `json:"vars"`
	Attachments []cliclient.AttachmentSpec `json:"attachments"`
}

// templatePositionRe extracts the line (and, when present, column) that
// text/template embeds in its own error strings, e.g.
// `template: tmpl:3: unexpected "}" in operand` or
// `template: tmpl:3:12: function "nope" not defined`. tmpl.Engine.Render
// returns only a plain wrapped error (internal/shell/tmpl/engine.go), so
// this is the only way to recover position info for the Preview pane's
// inline error display (SPEC.md §7.7.4.1: "never a bare 'render failed'").
var templatePositionRe = regexp.MustCompile(`template: tmpl:(\d+)(?::(\d+))?:`)

func parseTemplatePosition(err error) (line, col int, ok bool) {
	m := templatePositionRe.FindStringSubmatch(err.Error())
	if m == nil {
		return 0, 0, false
	}
	line, _ = strconv.Atoi(m[1])
	if m[2] != "" {
		col, _ = strconv.Atoi(m[2])
	}
	return line, col, true
}

// renderErrorResponse writes a render/send failure as a structured error
// including whatever position info parseTemplatePosition can recover.
func renderErrorResponse(c *gin.Context, status int, code string, err error) {
	body := gin.H{"code": code, "message": err.Error()}
	if line, col, ok := parseTemplatePosition(err); ok {
		body["line"] = line
		if col > 0 {
			body["column"] = col
		}
	}
	c.JSON(status, gin.H{"error": body})
}

// varsToData converts the request's string-keyed vars map into the
// map[string]any a tmpl.Engine.Render call expects as its data argument.
func varsToData(vars map[string]string) map[string]any {
	data := make(map[string]any, len(vars))
	for k, v := range vars {
		data[k] = v
	}
	return data
}

// renderEML renders text as a whole RFC 5322 document, matching
// internal/shell/builtin/send.go's modeTemplate handling.
func renderEML(engine *tmpl.Engine, text string, vars map[string]string) (string, error) {
	return engine.Render(text, varsToData(vars))
}

// renderJSONSpec unmarshals text into a cliclient.MessageSpec and templates
// each of its string fields individually, matching
// internal/shell/builtin/send.go's jsonSpec — the JSON document itself is
// not rendered as one big template, only the string values inside it.
func renderJSONSpec(engine *tmpl.Engine, text string, vars map[string]string) (cliclient.MessageSpec, error) {
	var spec cliclient.MessageSpec
	if err := json.Unmarshal([]byte(text), &spec); err != nil {
		return spec, fmt.Errorf("parsing json spec: %w", err)
	}

	data := varsToData(vars)
	render := func(v string) (string, error) { return engine.Render(v, data) }

	var err error
	if spec.From, err = render(spec.From); err != nil {
		return spec, err
	}
	if spec.Subject, err = render(spec.Subject); err != nil {
		return spec, err
	}
	if spec.Text, err = render(spec.Text); err != nil {
		return spec, err
	}
	if spec.HTML, err = render(spec.HTML); err != nil {
		return spec, err
	}
	if spec.To, err = renderAllStrings(render, spec.To); err != nil {
		return spec, err
	}
	if spec.Cc, err = renderAllStrings(render, spec.Cc); err != nil {
		return spec, err
	}
	if spec.Bcc, err = renderAllStrings(render, spec.Bcc); err != nil {
		return spec, err
	}
	if spec.Attachments, err = renderAttachments(render, spec.Attachments); err != nil {
		return spec, err
	}
	return spec, nil
}

// renderAttachments templates each attachment's Path and Filename — unlike
// internal/shell/builtin/send.go's jsonSpec (which leaves --json's
// Attachments untouched, since paths there come from the caller's own
// filesystem), the Composer's Attachments.Path is the only way to reference
// a template-generated file (e.g. {{ fPNG }}), since there is no --attach
// flag equivalent in an HTTP request body. The generating function's file
// lives in this request's tmpl.Engine tempDir, which stays alive until the
// handler returns — after spec.Build/cliclient.SendTLS have already read it.
func renderAttachments(render func(string) (string, error), in []cliclient.AttachmentSpec) ([]cliclient.AttachmentSpec, error) {
	out := make([]cliclient.AttachmentSpec, len(in))
	for i, att := range in {
		path, err := render(att.Path)
		if err != nil {
			return nil, err
		}
		filename, err := render(att.Filename)
		if err != nil {
			return nil, err
		}
		out[i] = cliclient.AttachmentSpec{Path: path, Filename: filename}
	}
	return out, nil
}

func renderAllStrings(render func(string) (string, error), in []string) ([]string, error) {
	out := make([]string, len(in))
	for i, v := range in {
		r, err := render(v)
		if err != nil {
			return nil, err
		}
		out[i] = r
	}
	return out, nil
}

// envelopeFromRendered derives the SMTP envelope from a rendered RFC 5322
// document's own From/To headers, matching
// internal/shell/builtin/send.go's envelopeFromMessage (minus its
// pflag-based --from/--to override, which has no equivalent here).
func envelopeFromRendered(raw string) (from string, to []string, err error) {
	msg, perr := mail.ReadMessage(strings.NewReader(raw))
	if perr == nil {
		if addrs, aerr := msg.Header.AddressList("From"); aerr == nil && len(addrs) > 0 {
			from = addrs[0].Address
		}
		if addrs, aerr := msg.Header.AddressList("To"); aerr == nil {
			for _, a := range addrs {
				to = append(to, a.Address)
			}
		}
	}
	if from == "" {
		return "", nil, fmt.Errorf("could not determine From (no From header in rendered message)")
	}
	if len(to) == 0 {
		return "", nil, fmt.Errorf("could not determine To (no To header in rendered message)")
	}
	return from, to, nil
}

// buildEMLWithAttachments splices attachments around rendered — an
// already-rendered whole-document EML template — producing a
// multipart/mixed envelope with the original headers preserved, the
// original body as the first part (under its original Content-Type, or
// text/plain if none was set), and one part per attachment. Returns
// rendered unchanged (as plain bytes) when there are no attachments, so
// non-attachment sends are byte-identical to before this feature existed.
func buildEMLWithAttachments(rendered string, attachments []cliclient.AttachmentSpec) ([]byte, error) {
	if len(attachments) == 0 {
		return []byte(rendered), nil
	}

	msg, err := mail.ReadMessage(strings.NewReader(rendered))
	if err != nil {
		return nil, fmt.Errorf("parsing rendered eml to attach files: %w", err)
	}
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return nil, fmt.Errorf("reading rendered eml body: %w", err)
	}

	bodyContentType := msg.Header.Get("Content-Type")
	if bodyContentType == "" {
		bodyContentType = "text/plain; charset=utf-8"
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	headerKeys := make([]string, 0, len(msg.Header))
	for k := range msg.Header {
		if k == "Content-Type" || k == "Mime-Version" {
			continue
		}
		headerKeys = append(headerKeys, k)
	}
	sort.Strings(headerKeys)
	for _, k := range headerKeys {
		for _, v := range msg.Header[k] {
			fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
		}
	}
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%q\r\n", mw.Boundary())
	buf.WriteString("\r\n")

	bodyWriter, err := mw.CreatePart(textproto.MIMEHeader{"Content-Type": {bodyContentType}})
	if err != nil {
		return nil, fmt.Errorf("creating eml body part: %w", err)
	}
	if _, err := bodyWriter.Write(body); err != nil {
		return nil, fmt.Errorf("writing eml body part: %w", err)
	}

	for _, att := range attachments {
		if err := cliclient.AttachFile(mw, att); err != nil {
			return nil, err
		}
	}

	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}
	return buf.Bytes(), nil
}

func renderHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req renderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
			return
		}

		engine, err := tmpl.New(0, false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "engine_init_failed", "message": err.Error()}})
			return
		}
		defer engine.Close()

		switch req.Format {
		case "eml":
			rendered, err := renderEML(engine, req.Template, req.Vars)
			if err != nil {
				renderErrorResponse(c, http.StatusBadRequest, "render_failed", err)
				return
			}

			resp := gin.H{"rendered": rendered}
			if len(req.Attachments) > 0 {
				data := varsToData(req.Vars)
				render := func(v string) (string, error) { return engine.Render(v, data) }
				attachments, err := renderAttachments(render, req.Attachments)
				if err != nil {
					renderErrorResponse(c, http.StatusBadRequest, "render_failed", err)
					return
				}
				resp["attachments"] = attachments
			}
			c.JSON(http.StatusOK, resp)

		case "json":
			spec, err := renderJSONSpec(engine, req.Template, req.Vars)
			if err != nil {
				renderErrorResponse(c, http.StatusBadRequest, "render_failed", err)
				return
			}
			rendered, err := json.MarshalIndent(spec, "", "  ")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "marshal_failed", "message": err.Error()}})
				return
			}
			c.JSON(http.StatusOK, gin.H{"rendered": string(rendered)})

		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_format", "message": fmt.Sprintf("format must be \"eml\" or \"json\", got %q", req.Format)}})
		}
	}
}
