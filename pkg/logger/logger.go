package logger

import (
	"log/slog"
	"os"
)

type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
}

type logger struct {
	*slog.Logger
}

func New(level string) Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return &logger{
		Logger: slog.New(handler),
	}
}

func (l *logger) Debug(msg string, args ...interface{}) {
	l.Logger.Debug(msg, args...)
}

func (l *logger) Info(msg string, args ...interface{}) {
	l.Logger.Info(msg, args...)
}

func (l *logger) Warn(msg string, args ...interface{}) {
	l.Logger.Warn(msg, args...)
}

func (l *logger) Error(msg string, args ...interface{}) {
	l.Logger.Error(msg, args...)
}

func (l *logger) Fatal(msg string, args ...interface{}) {
	l.Logger.Error(msg, args...)
	os.Exit(1)
}

