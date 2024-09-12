package utils

import "log/slog"

func HandleFatal(log *slog.Logger, msg string, errors ...error) {
	if msg == "" {
		msg = "fatal error"
	}
	for _, err := range errors {
		log.Error(msg, slog.String("error", err.Error()))
	}
}
