package logging

import (
	"go.uber.org/zap"
)

// NewLogger creates a zap.Logger based on the environment
func NewLogger(env Environment, service string) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	switch env {
	case ProductionEnvironment:

		logger, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	default:

		logger, err = zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
	}

	logger = logger.With(zap.String("service", service))

	// TODO: add discord hook

	return logger, nil
}
