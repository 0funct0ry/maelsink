// Package msgspec builds cliclient.MessageSpec values from randomized/
// scenario-seeded content (SPEC.md §7.6.2/§7.6.5) and provides the
// interval/jitter scheduling math §7.6.3 needs — the parts of
// internal/shell/builtin's randmsg/intmsg/weirdmsg/blast/deluge builtins
// that depend only on a *tmpl.Engine and plain data, not on *shell.Session
// or *pflag.FlagSet. Both internal/shell/builtin (the interactive shell) and
// internal/compose/job (compose's Jobs Panel, M13.3) build on this package
// so the send-loop/content-generation logic exists in exactly one place.
package msgspec

import (
	"fmt"

	"github.com/0funct0ry/maelsink/internal/cliclient"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// ContentParams mirrors SPEC.md §7.6.5's shared content-shaping flags,
// common to every one of randmsg/intmsg/weirdmsg/blast/deluge.
type ContentParams struct {
	To             string
	From           string
	Cc             []string
	Bcc            []string
	Subject        string
	Body           string // text|html|both|random ("" defaults to "random")
	Attachments    int
	AttachmentSize string // e.g. "10KB" ("" defaults to "10KB")
	Tags           []string
	Scenario       string
}

// BuildRandomSpec builds one cliclient.MessageSpec per SPEC.md §7.6.2's
// default table: every field defaults to a template-function call when its
// param is empty, so repeated calls (e.g. across a job's bulk/interval send
// loop) each produce distinct fake content from engine's seeded PRNG.
func BuildRandomSpec(engine *tmpl.Engine, data map[string]any, p ContentParams) (cliclient.MessageSpec, error) {
	render := func(expr string) (string, error) { return engine.Render(expr, data) }

	var spec cliclient.MessageSpec
	var err error

	to := p.To
	if to == "" {
		to = "{{ fEmail }}"
	}
	if spec.To, err = renderAll(render, []string{to}); err != nil {
		return spec, err
	}

	from := p.From
	if from == "" {
		from = "{{ fEmail }}"
	}
	if spec.From, err = render(from); err != nil {
		return spec, err
	}

	if spec.Cc, err = renderAll(render, p.Cc); err != nil {
		return spec, err
	}
	if spec.Bcc, err = renderAll(render, p.Bcc); err != nil {
		return spec, err
	}

	var scenario ExampleTemplate
	haveScenario := false
	if p.Scenario != "" {
		scenario, haveScenario = FindScenario(p.Scenario)
		if !haveScenario {
			return spec, fmt.Errorf("unknown scenario %q", p.Scenario)
		}
	}

	subject := p.Subject
	if subject == "" {
		if haveScenario {
			subject = scenario.Subject
		} else {
			subject = "{{ fSubject }}"
		}
	}
	if spec.Subject, err = render(subject); err != nil {
		return spec, err
	}

	if err := renderRandomBody(engine, data, p.Body, haveScenario, scenario, &spec); err != nil {
		return spec, err
	}

	if p.Attachments > 0 {
		size := p.AttachmentSize
		if size == "" {
			size = "10KB"
		}
		specs, err := randomAttachments(engine, data, p.Attachments, size)
		if err != nil {
			return spec, err
		}
		spec.Attachments = specs
	}

	spec.Tags = p.Tags

	return spec, nil
}

// renderRandomBody fills spec.Text/HTML per mode (text|html|both|random),
// preferring the scenario's own Text/HTML when haveScenario is set (falling
// back to fake-generated bodies for whichever of Text/HTML the scenario
// left empty, matching mode's setting for that field).
func renderRandomBody(engine *tmpl.Engine, data map[string]any, mode string, haveScenario bool, scenario ExampleTemplate, spec *cliclient.MessageSpec) error {
	render := func(expr string) (string, error) { return engine.Render(expr, data) }

	if mode == "" {
		mode = "random"
	}
	switch mode {
	case "text", "html", "both", "random":
	default:
		return fmt.Errorf("body must be text, html, both, or random, got %q", mode)
	}
	if mode == "random" {
		switch engine.Intn(3) {
		case 0:
			mode = "text"
		case 1:
			mode = "html"
		default:
			mode = "both"
		}
	}

	wantText := mode == "text" || mode == "both"
	wantHTML := mode == "html" || mode == "both"

	var err error
	if wantText {
		src := "{{ fTextBody }}"
		if haveScenario && scenario.Text != "" {
			src = scenario.Text
		}
		if spec.Text, err = render(src); err != nil {
			return err
		}
	}
	if wantHTML {
		src := "{{ fHTMLBody }}"
		if haveScenario && scenario.HTML != "" {
			src = scenario.HTML
		}
		if spec.HTML, err = render(src); err != nil {
			return err
		}
	}
	return nil
}

// randomAttachments generates n files of the given size via the template
// engine's fBinary function (through Render, since Engine's file-generating
// methods are private to the tmpl package) and returns them as
// AttachmentSpecs.
func randomAttachments(engine *tmpl.Engine, data map[string]any, n int, size string) ([]cliclient.AttachmentSpec, error) {
	specs := make([]cliclient.AttachmentSpec, 0, n)
	for i := 0; i < n; i++ {
		path, err := engine.Render(fmt.Sprintf("{{ fBinary %q }}", size), data)
		if err != nil {
			return nil, fmt.Errorf("generating attachment: %w", err)
		}
		specs = append(specs, cliclient.AttachmentSpec{Path: path})
	}
	return specs, nil
}

// renderAll renders every entry of in through render, preserving order.
func renderAll(render func(string) (string, error), in []string) ([]string, error) {
	if in == nil {
		return nil, nil
	}
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
