package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

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
