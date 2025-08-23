package devices

import (
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
)
