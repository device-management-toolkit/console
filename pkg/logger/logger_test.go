package logger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var mu sync.Mutex

type loggerTest struct {
	name       string
	logLevel   zerolog.Level
	logMessage string
}

const (
	levelError = "error"
	levelWarn  = "warn"
	levelInfo  = "info"
	levelDebug = "debug"
)

func TestLogger(t *testing.T) { //nolint:paralleltest // logging library is not thread-safe for tests
	tests := []loggerTest{
		{
			name:       "Debug level logging",
			logLevel:   zerolog.DebugLevel,
			logMessage: "debug message",
		},
		{
			name:       "Info level logging",
			logLevel:   zerolog.InfoLevel,
			logMessage: "info message",
		},
		{
			name:       "Warn level logging",
			logLevel:   zerolog.WarnLevel,
			logMessage: "warn message",
		},
		{
			name:       "Error level logging",
			logLevel:   zerolog.ErrorLevel,
			logMessage: "error message",
		},
	}

	for _, tc := range tests { //nolint:paralleltest // logging library is not thread-safe for tests
		tc := tc // capture range variable

		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			zl := zerolog.New(&buf).With().Timestamp().Logger().Level(tc.logLevel)

			log := logger{logger: &zl}

			log.Debug("debug message")
			log.Info("info message")
			log.Warn("warn message")
			log.Error("error message")

			switch strings.ToLower(tc.logLevel.String()) {
			case levelError:
				assert.Contains(t, buf.String(), tc.logMessage)
				assert.NotContains(t, buf.String(), "debug")
				assert.NotContains(t, buf.String(), "info")
				assert.NotContains(t, buf.String(), "warn")
			case levelWarn:
				assert.Contains(t, buf.String(), tc.logMessage)
				assert.Contains(t, buf.String(), "error")
				assert.NotContains(t, buf.String(), "debug")
				assert.NotContains(t, buf.String(), "info")
			case levelInfo:
				assert.Contains(t, buf.String(), tc.logMessage)
				assert.Contains(t, buf.String(), "error")
				assert.Contains(t, buf.String(), "warn")
				assert.NotContains(t, buf.String(), "debug")
			case levelDebug:
				assert.Contains(t, buf.String(), tc.logMessage)
				assert.Contains(t, buf.String(), "error")
				assert.Contains(t, buf.String(), "info")
				assert.Contains(t, buf.String(), "warn")
			}
		})
	}
}

func TestFatal(t *testing.T) {
	t.Parallel()

	zl := zerolog.New(os.Stdout).With().Timestamp().Logger().Level(zerolog.FatalLevel)

	log := &logger{logger: &zl}

	if os.Getenv("BE_CRASHER") == "1" {
		log.Fatal("fatal message")

		return
	}

	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestFatal") // #nosec
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")

	err := cmd.Run()

	var exitError *exec.ExitError
	if errors.As(err, &exitError) && !exitError.Success() {
		return
	}

	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestNewLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level         string
		expectedLevel zerolog.Level
	}{
		{levelDebug, zerolog.DebugLevel},
		{levelInfo, zerolog.InfoLevel},
		{levelWarn, zerolog.WarnLevel},
		{levelError, zerolog.ErrorLevel},
		{"invalid", zerolog.InfoLevel},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(fmt.Sprintf("LogLevel_%s", tc.level), func(t *testing.T) {
			t.Parallel()

			mu.Lock()
			defer mu.Unlock()

			log := New(tc.level)
			require.NotNil(t, log)

			level := log.(*logger).logger.GetLevel() //nolint:errcheck // It's a test...
			assert.Equal(t, tc.expectedLevel, level)
		})
	}
}
