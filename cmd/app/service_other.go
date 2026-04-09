//go:build !windows

package main

import (
	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var serviceMode bool

func isServiceMode() bool {
	return false
}

func runAsService(_ *config.Config, _ logger.Interface) error {
	return nil
}
