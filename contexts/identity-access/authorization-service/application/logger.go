package application

import "log/slog"

// ResolveLogger returns the provided logger or falls back to slog default.
func ResolveLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}
