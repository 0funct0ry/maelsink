package compose

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// sendHandler renders req the same way renderHandler does, then delivers
// the result over SMTP to the target configured in cfg — the same
// underlying cliclient.SendTLS path internal/shell/builtin's send command
// uses (SPEC.md §7.7.4.1's "same underlying path"). Fully stateless: no
// session or vars persist between calls.
func sendHandler(cfg TargetConfig) gin.HandlerFunc {
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

		var raw []byte
		var from string
		var to []string

		switch req.Format {
		case "eml":
			rendered, err := renderEML(engine, req.Template, req.Vars)
			if err != nil {
				renderErrorResponse(c, http.StatusBadRequest, "render_failed", err)
				return
			}
			from, to, err = envelopeFromRendered(rendered)
			if err != nil {
				renderErrorResponse(c, http.StatusBadRequest, "envelope_failed", err)
				return
			}

			var attachments []cliclient.AttachmentSpec
			if len(req.Attachments) > 0 {
				data := varsToData(req.Vars)
				render := func(v string) (string, error) { return engine.Render(v, data) }
				attachments, err = renderAttachments(render, req.Attachments)
				if err != nil {
					renderErrorResponse(c, http.StatusBadRequest, "render_failed", err)
					return
				}
			}
			raw, err = buildEMLWithAttachments(rendered, attachments)
			if err != nil {
				renderErrorResponse(c, http.StatusBadRequest, "build_failed", err)
				return
			}

		case "json":
			spec, err := renderJSONSpec(engine, req.Template, req.Vars)
			if err != nil {
				renderErrorResponse(c, http.StatusBadRequest, "render_failed", err)
				return
			}
			raw, err = spec.Build(time.Now())
			if err != nil {
				renderErrorResponse(c, http.StatusBadRequest, "build_failed", err)
				return
			}
			from, to = spec.Envelope()

		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_format", "message": "format must be \"eml\" or \"json\""}})
			return
		}

		var auth *cliclient.Auth
		if cfg.SMTPUser != "" || cfg.SMTPPass != "" {
			auth = &cliclient.Auth{Username: cfg.SMTPUser, Password: cfg.SMTPPass}
		}

		if err := cliclient.SendTLS(c.Request.Context(), cfg.SMTPAddr, cliclient.TLSOptions{}, auth, from, to, raw); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "smtp_send_failed", "message": err.Error()}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"from": from, "to": to})
	}
}
