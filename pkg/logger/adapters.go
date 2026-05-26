package logger

import (
	"bytes"
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

// logrusHook forwards logrus Trace-level entries to the console logger as Debug messages.
// This surfaces the WSMAN request/response XML that go-wsman-messages emits via logrus.Trace
// when LogAMTMessages is true.
type logrusHook struct {
	l Interface
}

func (h *logrusHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.TraceLevel}
}

func (h *logrusHook) Fire(entry *logrus.Entry) error {
	h.l.Debug("wsman: " + entry.Message)

	return nil
}

// SetupLogrus installs a logrus hook that forwards WSMAN trace messages (request/response XML)
// to the console logger at Debug level, and sets the logrus level to Trace so the messages
// are not dropped before reaching the hook.
// Call this only when the console log level is "debug".
func SetupLogrus(l Interface) {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.AddHook(&logrusHook{l: l})
}
