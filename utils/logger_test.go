package utils

import "testing"

func TestNewLogger_ValidLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			logger := NewLogger(level)
			if logger == nil {
				t.Errorf("expected logger for level %s, got nil", level)
			}
		})
	}
}

func TestNewLogger_UnknownLevel(t *testing.T) {
	logger := NewLogger("some-unknown-level")

	if logger == nil {
		t.Error("expected logger for unknown level (should default to info), got nil")
	}
}

func TestLogger_Methods(t *testing.T) {
	logger := NewLogger("debug")

	logger.Info("test info", "key", "value")
	logger.Error("test error")
	logger.Error("test error", "error", "something")
	logger.Debug("test debug")
}
