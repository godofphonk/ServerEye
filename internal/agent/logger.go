package agent

import (
	"github.com/godofphonk/ServerEye/internal/config"
	"github.com/sirupsen/logrus"
)

// setupLoggerFromConfig configures logger based on configuration
func setupLoggerFromConfig(cfg *config.AgentConfig, logger *logrus.Logger) {
	// Set log level from config
	logLevel, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Set formatter
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
}
