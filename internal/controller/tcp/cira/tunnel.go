package cira

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/apf"
	wsman2 "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/client"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

const (
	maxIdleTime = 300 * time.Second
	port        = "4433"
)

var mu sync.Mutex

type Server struct {
	certificates tls.Certificate
	notify       chan error
	listener     net.Listener
	devices      devices.Feature
}

func NewServer(certFile, keyFile string, d devices.Feature) (*Server, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	s := &Server{
		certificates: cert,
		notify:       make(chan error, 1),
		devices:      d,
	}

	s.start()

	return s, nil
}

func (s *Server) start() {
	go func() {
		s.notify <- s.ListenAndServe()
		close(s.notify)
	}()
}

// Notify returns the error channel for server notifications.
func (s *Server) Notify() <-chan error {
	return s.notify
}

func (s *Server) ListenAndServe() error {
	config := &tls.Config{
		Certificates:       []tls.Certificate{s.certificates},
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}

	defaultCipherSuites := tls.CipherSuites()
	config.CipherSuites = make([]uint16, 0, len(defaultCipherSuites)+3)

	for _, suite := range defaultCipherSuites {
		config.CipherSuites = append(config.CipherSuites, suite.ID)
	}
	// add the weak cipher suites for AMT device compatibility
	config.CipherSuites = append(config.CipherSuites,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	)

	listener, err := tls.Listen("tcp", ":"+port, config)
	if err != nil {
		return err
	}

	s.listener = listener

	log.Printf("Server running on port %s\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		log.Println("Failed to cast connection to TLS connection")

		return
	}

	log.Println("New TLS connection detected")

	// Create handler with access to devices feature for credential validation
	handler := NewAPFHandler(s.devices)

	// Create processor with our handler - all decoding happens in the APF library
	processor := apf.NewProcessor(handler)

	// Initialize session
	session := &apf.Session{}

	var device *wsman.ConnectionEntry

	authenticated := false

	defer func() {
		deviceID := handler.DeviceID()
		if authenticated && deviceID != "" {
			mu.Lock()
			delete(wsman.Connections, deviceID)
			mu.Unlock()
		}
	}()

	for {
		if err := conn.SetDeadline(time.Now().Add(maxIdleTime)); err != nil {
			log.Printf("Failed to set deadline: %v\n", err)

			break
		}

		buf := make([]byte, 4096)

		n, err := tlsConn.Read(buf)
		if err != nil && n == 0 {
			deviceID := handler.DeviceID()
			if errors.Is(err, net.ErrClosed) {
				log.Printf("Connection closed for device %s\n", deviceID)

				break
			}

			log.Printf("Read error for device %s: %v\n", deviceID, err)

			break
		}

		data := buf[:n]
		log.Printf("Received data: %s\n", hex.EncodeToString(data))

		// Store message type before processing
		messageType := byte(0)
		if len(data) > 0 {
			messageType = data[0]
		}

		// Process through APF processor - all decoding and auth validation happens here
		response := processor.Process(data, session)

		// Handle authentication flow
		if messageType == apf.APF_USERAUTH_REQUEST && !authenticated {
			responseBytes := response.Bytes()
			// Check if authentication succeeded by examining response
			// Success response is just 1 byte (APF_USERAUTH_SUCCESS = 52)
			if len(responseBytes) > 0 && responseBytes[0] == apf.APF_USERAUTH_SUCCESS {
				authenticated = true
				deviceID := handler.DeviceID()

				clientParams := client.Parameters{}
				device = &wsman.ConnectionEntry{
					IsCIRA:        true,
					Conny:         conn,
					Timer:         time.NewTimer(maxIdleTime),
					WsmanMessages: wsman2.NewMessages(clientParams),
				}

				mu.Lock()
				wsman.Connections[deviceID] = device
				mu.Unlock()

				log.Printf("Device authenticated and registered with ID: %s\n", deviceID)
			} else {
				// Authentication failed - send response and close connection
				_, _ = conn.Write(responseBytes)
				log.Printf("Authentication failed for device, closing connection\n")

				break
			}
		}

		// Write response
		_, err = conn.Write(response.Bytes())
		if err != nil {
			log.Printf("Write error for device %s: %v\n", handler.DeviceID(), err)

			break
		}

		// Send keep-alive options if handler indicates it's time
		if authenticated && messageType == apf.APF_GLOBAL_REQUEST && handler.ShouldSendKeepAlive() {
			var binBuf bytes.Buffer

			// TODO: Make these values configurable from console config
			keepAliveOptionsRequest := apf.KeepAliveOptionsRequest(30, 90)

			err := binary.Write(&binBuf, binary.BigEndian, keepAliveOptionsRequest)
			if err != nil {
				log.Printf("Error creating keep-alive request: %v\n", err)

				continue
			}

			_, err = conn.Write(binBuf.Bytes())
			if err != nil {
				log.Printf("Error sending keep-alive request: %v\n", err)

				break
			}

			log.Printf("Sent keep-alive options request for device %s\n", handler.DeviceID())
		}
	}
}

// Shutdown gracefully shuts down the CIRA server.
func (s *Server) Shutdown() error {
	if s.listener != nil {
		return s.listener.Close()
	}

	return nil
}
