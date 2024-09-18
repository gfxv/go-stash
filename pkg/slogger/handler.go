package slogger

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
)

type SloggerHandlerOpts struct {
	SlogOpts *slog.HandlerOptions
}

type SloggerHandler struct {
	slog.Handler
	l *log.Logger
}

func (h *SloggerHandler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String() + ":"
	fields := make(map[string]interface{}, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.Any()
		return true
	})
	b, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return err
	}
	timeStr := r.Time.Format("[15:04:05]")
	msg := r.Message

	h.l.Println(timeStr, level, msg, string(b))

	return nil
}

func NewSloggerHandler(out io.Writer, opts SloggerHandlerOpts) *SloggerHandler {
	h := &SloggerHandler{
		Handler: slog.NewJSONHandler(out, opts.SlogOpts),
		l:       log.New(out, "", 0),
	}
	return h
}
