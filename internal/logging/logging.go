// Package logging provides the shared structured logger for maelsink.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// New builds a slog.Logger per the resolved logging config (SPEC.md §3.1).
//
// format selects the handler explicitly and is never auto-selected — it
// must come from logging.format / --log-format, defaulting to "text"
// upstream in internal/config. file, when non-empty, additionally writes
// log output to that path; an empty file means stdout only.
func New(level, format, file string) (*slog.Logger, error) {
	var out io.Writer = os.Stdout
	if file != "" {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("logging: open log file %q: %w", file, err)
		}
		out = io.MultiWriter(os.Stdout, f)
	}

	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var handler slog.Handler
	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(out, opts)
	case "text", "":
		handler = slog.NewTextHandler(out, opts)
	default:
		return nil, fmt.Errorf("logging: unknown log format %q (want text|json)", format)
	}

	return slog.New(handler), nil
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
