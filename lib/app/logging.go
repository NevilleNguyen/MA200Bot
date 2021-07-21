package app

import (
	"fmt"

	"go.uber.org/zap"
)

type syncer interface {
	Sync() error
}

// NewFlusher creates a new syncer from given syncer that log a error message if failed to sync.
func NewFlusher(s syncer) func() {
	return func() {
		// ignore the error as the sync function will always fail in Linux
		// https://github.com/uber-go/zap/issues/370
		_ = s.Sync()
	}
}

// NewLogger creates a new logger instance, input is context
func NewLogger(mode string) (*zap.Logger, error) {

	switch mode {
	case "production":
		return zap.NewProduction()
	case "development":
		return zap.NewDevelopment()
	default:
		return nil, fmt.Errorf("invalid running mode: %q", mode)
	}
}

// NewSugaredLogger creates a new sugared logger and a flush function. The flush function should be
// called by consumer before quitting application.
// This function should be use most of the time unless
// the application requires extensive performance, in this case use NewLogger.
func NewSugaredLogger() (*zap.SugaredLogger, func(), error) {
	logger, err := NewLogger("development")
	if err != nil {
		return nil, nil, err
	}

	sugar := logger.Sugar()
	return sugar, NewFlusher(logger), nil
}
