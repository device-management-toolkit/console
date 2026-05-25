//go:build tray

package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/pkg/tray"
)

var trayBuildEnabled = true

// isTerminal returns true if stdin is connected to a terminal.
func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return fi.Mode()&os.ModeCharDevice != 0
}

func runWithTray(cfg *config.Config, l logger.Interface) {
	urls := listenURLs(cfg)
	primaryURL := urls[0]

	// Must run before re-exec so the parent surfaces the duplicate, not the soon-to-exit child.
	ensureSingleInstance(primaryURL)

	if os.Getenv("DMT_BACKGROUND") == "" && isTerminal() {
		// Print before re-exec; after fork, stderr is the log file.
		for _, u := range urls {
			log.Printf("DMT Console running at %s", u)
		}

		relaunchInBackground()
	}

	trayManager := tray.New(tray.Config{
		AppName:  "DMT Console",
		URL:      primaryURL,
		Headless: isHeadlessBuild,
		OnReady: func() {
			go runAppFunc(cfg, l)

			for _, u := range urls {
				log.Printf("DMT Console running at %s", u)
			}
		},
		OnQuit: func() {
			log.Println("Shutting down DMT Console...")
			// Send interrupt signal to trigger graceful shutdown
			p, err := os.FindProcess(os.Getpid())
			if err != nil {
				log.Printf("Failed to find current process for shutdown signal: %v", err)

				return
			}

			if err := p.Signal(os.Interrupt); err != nil {
				log.Printf("Failed to send interrupt signal: %v", err)
			}
		},
	})

	// Catch Ctrl+C / SIGTERM so the tray unblocks on terminal interrupt.
	// app.Run also listens for these signals to shut down the HTTP server;
	// Go delivers to all registered channels, so both handlers fire.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		trayManager.Quit()
	}()

	// Run the tray (this blocks until quit)
	trayManager.Run()
}

// listenURLs returns reachable URLs; the first entry is the tray's "open in browser" target.
func listenURLs(cfg *config.Config) []string {
	scheme := "http"
	if cfg.TLS.Enabled {
		scheme = "https"
	}

	hosts := listenHosts(cfg.Host)
	urls := make([]string, 0, len(hosts))

	for _, h := range hosts {
		urls = append(urls, scheme+"://"+net.JoinHostPort(unbracketHost(h), cfg.Port))
	}

	return urls
}

// listenHosts always returns at least one entry ("localhost") so listenURLs[0] is safe.
func listenHosts(cfgHost string) []string {
	if !isWildcardListenHost(cfgHost) {
		return []string{cfgHost}
	}

	hosts := []string{addrLocalhost}

	ifaces, err := net.Interfaces()
	if err != nil {
		return hosts
	}

	seen := map[string]struct{}{}

	for _, iface := range ifaces {
		if !isReachableInterface(iface) {
			continue
		}

		hosts = appendInterfaceIPv4s(hosts, iface, seen)
	}

	return hosts
}

func isReachableInterface(iface net.Interface) bool {
	if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
		return false
	}

	return !isVirtualInterfaceName(iface.Name)
}

func appendInterfaceIPv4s(hosts []string, iface net.Interface, seen map[string]struct{}) []string {
	addrs, err := iface.Addrs()
	if err != nil {
		return hosts
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		if ipNet.IP.IsLoopback() || ipNet.IP.IsLinkLocalUnicast() || ipNet.IP.To4() == nil {
			continue
		}

		ip := ipNet.IP.String()
		if _, dup := seen[ip]; dup {
			continue
		}

		seen[ip] = struct{}{}
		hosts = append(hosts, ip)
	}

	return hosts
}

// isVirtualInterfaceName filters out container/VM/VPN bridges that bind addresses
// the user can't actually reach the tray from. Conservative — we'd rather miss
// a real NIC than spam the user with a dozen 172.17.x.x docker URLs.
func isVirtualInterfaceName(name string) bool {
	prefixes := []string{
		"docker", "br-", "veth",
		"tun", "tap", "utun",
		"virbr", "vmnet", "vboxnet",
		"awdl", "llw",
		"zt", "wg",
	}

	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}

	return false
}
