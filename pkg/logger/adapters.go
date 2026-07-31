package logger

import (
	"bytes"
	"io"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type adapterLevel int

const (
	adapterLevelInfo adapterLevel = iota
	adapterLevelWarn
	adapterLevelError
)

// writerAdapter implements io.Writer and forwards messages to our logger.
type writerAdapter struct {
	l     Interface
	level adapterLevel
}

func (w writerAdapter) Write(p []byte) (n int, err error) {
	msg := bytes.TrimRight(p, "\r\n")

	switch w.level {
	case adapterLevelInfo:
		w.l.Info(string(msg))
	case adapterLevelWarn:
		w.l.Warn(string(msg))
	case adapterLevelError:
		w.l.Error(string(msg))
	}

	return len(p), nil
}

// SetupStdLog routes the standard library log output through our JSON logger.
func SetupStdLog(l Interface) {
	log.SetFlags(0)
	log.SetOutput(writerAdapter{l: l, level: adapterLevelWarn})
}

// SetupGin routes Gin's logs through our JSON logger.
func SetupGin(l Interface) {
	gin.DefaultWriter = writerAdapter{l: l, level: adapterLevelInfo}
	gin.DefaultErrorWriter = writerAdapter{l: l, level: adapterLevelError}
}

// logrusHook forwards selected logrus entries through the console logger so
// formatting is consistent with all other console logs.
type logrusHook struct {
	l Interface
}

func (h *logrusHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.DebugLevel,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
	}
}

func (h *logrusHook) Fire(entry *logrus.Entry) error {
	switch entry.Level {
	case logrus.DebugLevel:
		h.l.Debug(entry.Message)
	case logrus.InfoLevel:
		h.l.Info(entry.Message)
	case logrus.WarnLevel:
		h.l.Warn(entry.Message)
	case logrus.ErrorLevel:
		h.l.Error(entry.Message)
	}

	return nil
}

// SetupLogrus installs a logrus hook and disables default logrus output so
// forwarded entries appear only in console logger format.
func SetupLogrus(l Interface) {
	logrus.SetOutput(io.Discard)
	logrus.AddHook(&logrusHook{l: l})
}
