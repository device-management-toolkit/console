package devices

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	kvmDeviceToBrowserBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kvm_device_to_browser_bytes_total",
			Help: "Total bytes forwarded from AMT device to browser (per mode)",
		},
		[]string{"mode"},
	)

	kvmBrowserToDeviceBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kvm_browser_to_device_bytes_total",
			Help: "Total bytes forwarded from browser to AMT device (per mode)",
		},
		[]string{"mode"},
	)

	kvmDeviceToBrowserMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kvm_device_to_browser_messages_total",
			Help: "Number of frames/messages from AMT device forwarded to browser (per mode)",
		},
		[]string{"mode"},
	)

	kvmBrowserToDeviceMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kvm_browser_to_device_messages_total",
			Help: "Number of frames/messages from browser forwarded to AMT device (per mode)",
		},
		[]string{"mode"},
	)

	kvmDeviceToBrowserWriteSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kvm_device_to_browser_write_seconds",
			Help:    "Time to write a device frame to the websocket (per mode)",
			Buckets: []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1},
		},
		[]string{"mode"},
	)

	kvmBrowserToDeviceSendSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kvm_browser_to_device_send_seconds",
			Help:    "Time to send a browser frame to the device TCP connection (per mode)",
			Buckets: []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1},
		},
		[]string{"mode"},
	)

	kvmDevicePayloadBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kvm_device_payload_bytes",
			Help:    "Distribution of device payload sizes forwarded to browser (per mode)",
			Buckets: []float64{64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
		},
		[]string{"mode"},
	)

	kvmBrowserPayloadBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kvm_browser_payload_bytes",
			Help:    "Distribution of browser payload sizes forwarded to device (per mode)",
			Buckets: []float64{64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
		},
		[]string{"mode"},
	)

	// Time spent blocked waiting for data.
	kvmDeviceReceiveBlockSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kvm_device_receive_block_seconds",
			Help:    "Time blocked on device TCP Receive() waiting for data (per mode)",
			Buckets: []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2},
		},
		[]string{"mode"},
	)

	kvmBrowserReadBlockSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kvm_browser_read_block_seconds",
			Help:    "Time blocked on websocket ReadMessage() from browser (per mode)",
			Buckets: []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2},
		},
		[]string{"mode"},
	)

	// KVM Connection Performance Metrics.
	kvmDeviceLookupSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "kvm_device_lookup_seconds",
			Help:    "Time to look up device from database during KVM connection (KVM_TIMING)",
			Buckets: []float64{0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2},
		},
	)

	kvmConnectionSetupSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kvm_connection_setup_seconds",
			Help:    "Time to establish TCP connection to device during KVM setup (KVM_TIMING)",
			Buckets: []float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10},
		},
		[]string{"mode"},
	)

	kvmWebsocketUpgradeSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "kvm_websocket_upgrade_seconds",
			Help:    "Time to upgrade HTTP connection to WebSocket for KVM (KVM_TIMING)",
			Buckets: []float64{0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1},
		},
	)

	kvmTotalConnectionSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kvm_total_connection_seconds",
			Help:    "Total time from request to ready KVM connection (KVM_TIMING)",
			Buckets: []float64{0.1, 0.2, 0.5, 1, 2, 5, 10, 20, 30},
		},
		[]string{"mode"},
	)

	kvmConsentCodeWaitSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kvm_consent_code_wait_seconds",
			Help:    "Time spent waiting for consent code handling during KVM setup (KVM_TIMING)",
			Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"mode"},
	)
)

// RecordWebsocketUpgrade records the WebSocket upgrade duration metric.
func RecordWebsocketUpgrade(duration time.Duration) {
	kvmWebsocketUpgradeSeconds.Observe(duration.Seconds())
}

// RecordTotalConnection records the total KVM connection time metric.
func RecordTotalConnection(duration time.Duration, mode string) {
	kvmTotalConnectionSeconds.WithLabelValues(mode).Observe(duration.Seconds())
}

// RecordConsentCodeWait records the consent code wait time metric.
func RecordConsentCodeWait(duration time.Duration, mode string) {
	kvmConsentCodeWaitSeconds.WithLabelValues(mode).Observe(duration.Seconds())
}

// RecordDeviceLookup records the device lookup duration metric.
func RecordDeviceLookup(duration time.Duration) {
	kvmDeviceLookupSeconds.Observe(duration.Seconds())
}

// RecordConnectionSetup records the TCP connection setup duration metric.
func RecordConnectionSetup(duration time.Duration, mode string) {
	kvmConnectionSetupSeconds.WithLabelValues(mode).Observe(duration.Seconds())
}
