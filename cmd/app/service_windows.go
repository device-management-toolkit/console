package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/sys/windows/svc"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// serviceMode is set early in main() to indicate we're running under the SCM.
var serviceMode bool

// isServiceMode detects whether the process was started by the Windows SCM.
func isServiceMode() bool {
	is, err := svc.IsWindowsService()
	if err != nil {
		log.Printf("Warning: could not detect service mode: %v", err)

		return false
	}

	return is
}

// consoleService implements svc.Handler for the Windows Service Control Manager.
type consoleService struct {
	cfg *config.Config
	log logger.Interface
}

func (s *consoleService) Execute(_ []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown
	status <- svc.Status{State: svc.StartPending}

	// Start the application in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		runAppFunc(s.cfg, s.log)
	}()

	status <- svc.Status{State: svc.Running, Accepts: accepted}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				// Signal the app to shut down via the same mechanism it uses
				p, _ := os.FindProcess(os.Getpid())
				_ = p.Signal(os.Interrupt)
				<-done
				return false, 0
			}
		case <-done:
			// App exited on its own
			return false, 0
		}
	}
}

// runAsService runs the app under the Windows SCM service handler.
func runAsService(cfg *config.Config, l logger.Interface) error {
	err := svc.Run("DMTConsole", &consoleService{cfg: cfg, log: l})
	if err != nil {
		return fmt.Errorf("running windows service: %w", err)
	}

	return nil
}
