package application

import "log/slog"

// ResolveLogger guarantees a non-nil logger for application/worker code paths.
func ResolveLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}
