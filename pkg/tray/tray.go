package tray

import (
	"os/exec"
	"runtime"

	"fyne.io/systray"
)

// Config holds the configuration for the system tray.
type Config struct {
	AppName  string
	URL      string
	Headless bool
	OnQuit   func()
	OnReady  func()
}

// Manager handles the system tray lifecycle.
type Manager struct {
	config Config
}

// New creates a new tray manager.
func New(cfg Config) *Manager {
	return &Manager{config: cfg}
}

// Run starts the system tray - this blocks until quit.
func (m *Manager) Run() {
	systray.Run(m.onReady, nil)
}

// Quit exits the system tray.
func (m *Manager) Quit() {
	systray.Quit()
}

func (m *Manager) onReady() {
	// SetTemplateIcon makes macOS treat it as a template image so it auto-inverts
	// for light/dark menu bars; on Windows/Linux it falls back to a normal icon.
	systray.SetTemplateIcon(getIcon(), getIcon())
	systray.SetTooltip(m.config.AppName)

	if m.config.Headless {
		m.onReadyHeadless()
	} else {
		m.onReadyFull()
	}
}

func (m *Manager) onReadyFull() {
	// Menu items
	mOpen := systray.AddMenuItem("Open "+m.config.AppName, "Open the web interface")

	systray.AddSeparator()

	mStatus := systray.AddMenuItem("Running on "+m.config.URL, "Server status")
	mStatus.Disable()
	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Stop the server and exit")

	// Call the onReady callback if provided
	if m.config.OnReady != nil {
		m.config.OnReady()
	}

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				_ = openBrowser(m.config.URL)
			case <-mQuit.ClickedCh:
				if m.config.OnQuit != nil {
					m.config.OnQuit()
				}

				systray.Quit()

				return
			}
		}
	}()
}

func (m *Manager) onReadyHeadless() {
	mMode := systray.AddMenuItem("Running in headless mode", "No web UI available")
	mMode.Disable()
	systray.AddSeparator()

	mStatus := systray.AddMenuItem("API on "+m.config.URL, "Server status")
	mStatus.Disable()
	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Stop the server and exit")

	// Call the onReady callback if provided
	if m.config.OnReady != nil {
		m.config.OnReady()
	}

	// Handle menu clicks
	go func() {
		<-mQuit.ClickedCh

		if m.config.OnQuit != nil {
			m.config.OnQuit()
		}

		systray.Quit()
	}()
}

// openBrowser opens the default browser to the given URL.
func openBrowser(url string) error {
	var (
		cmd  string
		args []string
	)

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		// rundll32 invokes the URL handler directly, avoiding cmd.exe's
		// metacharacter parsing that affects "cmd /c start <url>".
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// getIcon returns the icon bytes for the system tray.
func getIcon() []byte {
	return iconData
}
