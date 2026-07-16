package utils

import (
	"log/slog"
	"os"
)

type Logger struct {
	logger *slog.Logger
}

func NewLogger(level string) *Logger {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug

	case "warn":
		slogLevel = slog.LevelWarn

	case "error":
		slogLevel = slog.LevelError

	default:
		slogLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	})

	return &Logger{
		logger: slog.New(handler),
	}
}

func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l Logger) Error(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *Logger) Fatal(msg string, args ...any) {
	l.logger.Error(msg, args...)
	os.Exit(1)
}
