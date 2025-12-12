/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package logger

import (
	"bytes"
	"log"

	"github.com/gin-gonic/gin"
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
