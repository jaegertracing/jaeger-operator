package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// InitLog creates a logger and enables the verbose level if specified
// enableVerbose: verbose logging level when true
func InitLog(enableVerbose bool) *logrus.Logger {
	log := logrus.New()
	log.Out = os.Stdout
	if enableVerbose {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	return log
}
