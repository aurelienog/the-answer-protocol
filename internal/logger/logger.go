package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

// levelSplitHandler routes INFO records to stdout and WARN/ERROR records to stderr.
// Concurrency safety is handled internally by slog.JSONHandler.
type levelSplitHandler struct {
	out slog.Handler
	err slog.Handler
}

func (h *levelSplitHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if level >= slog.LevelWarn {
		return h.err.Enabled(ctx, level)
	}
	return h.out.Enabled(ctx, level)
}

func (h *levelSplitHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelWarn {
		return h.err.Handle(ctx, r)
	}
	return h.out.Handle(ctx, r)
}

func (h *levelSplitHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelSplitHandler{
		out: h.out.WithAttrs(attrs),
		err: h.err.WithAttrs(attrs),
	}
}

func (h *levelSplitHandler) WithGroup(name string) slog.Handler {
	return &levelSplitHandler{
		out: h.out.WithGroup(name),
		err: h.err.WithGroup(name),
	}
}

// New creates a structured JSON logger writing INFO to stdout and WARN/ERROR to stderr.
func New() *slog.Logger {
	return NewWithWriters(os.Stdout, os.Stderr)
}

// NewWithWriters creates a structured JSON logger with custom output streams (useful for tests).
func NewWithWriters(stdout, stderr io.Writer) *slog.Logger {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			switch attr.Key {
			case slog.TimeKey:
				return slog.String(
					"timestamp",
					attr.Value.Time().UTC().Format(time.RFC3339Nano),
				)
			case slog.MessageKey:
				attr.Key = "event"
				return attr
			default:
				return attr
			}
		},
	}

	return slog.New(&levelSplitHandler{
		out: slog.NewJSONHandler(stdout, opts),
		err: slog.NewJSONHandler(stderr, opts),
	})
}

// NewNop returns a silent logger safe for unit tests or nil fallbacks.
func NewNop() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}