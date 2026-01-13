package logger_config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Logger is the shared structured logger.
// It is safe for concurrent use.
var Logger *slog.Logger

func init() {
	level := parseLevel(os.Getenv("LOG_LEVEL")) // debug|info|warn|error
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true, // file:line
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	Logger = slog.New(handler)

	// Make it the default logger (optional but convenient).
	slog.SetDefault(Logger)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

// Sugar helpers (printf-style), convenient for quick telemetry.
func Debugf(format string, args ...any) { Logger.Debug(fmt.Sprintf(format, args...)) }
func Infof(format string, args ...any)  { Logger.Info(fmt.Sprintf(format, args...)) }
func Warnf(format string, args ...any)  { Logger.Warn(fmt.Sprintf(format, args...)) }
func Errorf(format string, args ...any) { Logger.Error(fmt.Sprintf(format, args...)) }
