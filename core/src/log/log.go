package log

import (
	"github.com/sirupsen/logrus"
)

type Fields logrus.Fields

var Logger *logrus.Logger

func init() {
	Logger = logrus.New()
}

func LogFields(fields Fields) *logrus.Entry {
	return Logger.WithFields(logrus.Fields(fields))
}
