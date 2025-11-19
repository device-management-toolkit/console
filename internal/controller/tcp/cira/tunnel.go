package cira

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/apf"
	wsman2 "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/client"
)

const (
	maxIdleTime = 300 * time.Second
	port        = "4433"
)

var mu sync.Mutex

type ConnectedDevice struct {
	// Add necessary fields for connected device
}

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

// Notify -.
func (s *Server) Notify() <-chan error {
	return s.notify
}

func (s *Server) ListenAndServe() error {
	config := &tls.Config{
		Certificates: []tls.Certificate{s.certificates},
		// ClientAuth:         tls.NoClientCert,
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

	// Initialize session - device will be created after successful authentication
	session := &apf.Session{}
	var deviceID string
	var device *wsman.ConnectionEntry
	authenticated := false

	defer func() {
		if authenticated && deviceID != "" {
			mu.Lock()
			delete(wsman.Connections, deviceID)
			mu.Unlock()
		}
	}()

	for {
		conn.SetDeadline(time.Now().Add(maxIdleTime))

		buf := make([]byte, 4096)

		n, err := tlsConn.Read(buf)
		if err != nil && n == 0 {
			if errors.Is(err, net.ErrClosed) {
				log.Printf("Connection closed for device %s\n", deviceID)

				break
			}

			log.Printf("Read error for device %s: %v\n", deviceID, err)

			break
		}

		data := buf[:n]

		newDeviceID, authSuccess, shouldDisconnect, err := s.processData(tlsConn, data, session, deviceID, authenticated)
		if err != nil {
			log.Printf("Data processing error for device %s: %v\n", deviceID, err)
			break
		}

		// If authentication failed, close the connection
		if shouldDisconnect {
			log.Printf("Authentication failed for device, closing connection\n")
			break
		}

		// If authentication succeeded, initialize the device connection
		if authSuccess && !authenticated {
			authenticated = true

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
		}

		// If we got a new device ID from protocol version message, update it
		if newDeviceID != "" && newDeviceID != deviceID {
			if authenticated {
				mu.Lock()
				delete(wsman.Connections, deviceID)
				deviceID = newDeviceID
				wsman.Connections[deviceID] = device
				mu.Unlock()
				log.Printf("Device ID updated to: %s\n", deviceID)
			} else {
				// Store the UUID but don't register yet
				deviceID = newDeviceID
				log.Printf("Device UUID received: %s (awaiting authentication)\n", deviceID)
			}
		}
	}
}

func (s *Server) processData(conn net.Conn, data []byte, session *apf.Session, currentDeviceID string, authenticated bool) (deviceID string, authSuccess, shouldDisconnect bool, err error) {
	log.Printf("Received data: %s\n", hex.EncodeToString(data))

	// Store message type before processing
	messageType := byte(0)
	if len(data) > 0 {
		messageType = data[0]
	}

	// Let apf.Process handle all message processing
	response := apf.Process(data, session)

	// After processing, check what type of message it was and extract relevant data
	var newDeviceID string

	// Check if this was a protocol version message - extract UUID from original data
	if messageType == apf.APF_PROTOCOLVERSION {
		newDeviceID = s.extractUUID(data)
	}

	// Check if this was an authentication request and validate credentials
	if !authenticated && messageType == apf.APF_USERAUTH_REQUEST {
		// Parse the authentication details from the original data
		username, password, parseErr := s.parseAuthRequest(data)
		if parseErr != nil {
			log.Printf("Failed to parse authentication request: %v\n", parseErr)
			// Send authentication failure before closing
			failureMsg := []byte{apf.APF_USERAUTH_FAILURE}
			conn.Write(failureMsg)
			return "", false, true, nil
		}

		// Validate credentials against database
		log.Printf("Authentication attempt - Device: %s, Username: %s\n", currentDeviceID, username)
		isValid := s.validateCredentials(currentDeviceID, username, password)

		if !isValid {
			log.Printf("Authentication failed for device %s with username %s\n", currentDeviceID, username)
			// Send authentication failure before closing
			failureMsg := []byte{apf.APF_USERAUTH_FAILURE}
			conn.Write(failureMsg)
			return "", false, true, nil
		}

		log.Printf("Authentication successful for device %s\n", currentDeviceID)
		// apf.Process already generated the success response, just write it
		_, err = conn.Write(response.Bytes())
		return newDeviceID, true, false, err
	}

	// Write the response from apf.Process
	_, err = conn.Write(response.Bytes())
	if err != nil {
		return "", false, false, err
	}

	return newDeviceID, false, false, nil
}

func (s *Server) parseAuthRequest(data []byte) (username, password string, err error) {
	if len(data) < 5 {
		return "", "", fmt.Errorf("data too short")
	}

	reader := bytes.NewReader(data[1:]) // Skip message type

	// Read username length
	var usernameLen uint32
	if err := binary.Read(reader, binary.BigEndian, &usernameLen); err != nil {
		return "", "", fmt.Errorf("failed to read username length: %w", err)
	}

	if usernameLen > 256 {
		return "", "", fmt.Errorf("username too long: %d", usernameLen)
	}

	// Read username
	usernameBytes := make([]byte, usernameLen)
	if _, err := reader.Read(usernameBytes); err != nil {
		return "", "", fmt.Errorf("failed to read username: %w", err)
	}
	username = string(usernameBytes)

	// Read service name length and skip service name
	var serviceLen uint32
	if err := binary.Read(reader, binary.BigEndian, &serviceLen); err != nil {
		return "", "", fmt.Errorf("failed to read service length: %w", err)
	}
	reader.Seek(int64(serviceLen), 1) // Skip service name

	// Read method name length and method name
	var methodLen uint32
	if err := binary.Read(reader, binary.BigEndian, &methodLen); err != nil {
		return "", "", fmt.Errorf("failed to read method length: %w", err)
	}
	methodBytes := make([]byte, methodLen)
	if _, err := reader.Read(methodBytes); err != nil {
		return "", "", fmt.Errorf("failed to read method: %w", err)
	}
	method := string(methodBytes)

	// If method is "password", read the password
	if method == "password" {
		var passwordLen uint32
		if err := binary.Read(reader, binary.BigEndian, &passwordLen); err != nil {
			return "", "", fmt.Errorf("failed to read password length: %w", err)
		}

		if passwordLen > 256 {
			return "", "", fmt.Errorf("password too long: %d", passwordLen)
		}

		passwordBytes := make([]byte, passwordLen)
		if _, err := reader.Read(passwordBytes); err != nil {
			return "", "", fmt.Errorf("failed to read password: %w", err)
		}
		password = string(passwordBytes)
	}

	return username, password, nil
}

func (s *Server) validateCredentials(deviceID, username, password string) bool {
	// TODO: Implement actual database validation
	// This should query the database to validate the credentials
	// For now, returning true for testing purposes
	//
	// Example implementation:
	// ctx := context.Background()
	// device, err := s.devices.GetDevice(ctx, deviceID)
	// if err != nil {
	//     return false
	// }
	// return device.Username == username && device.Password == password

	log.Printf("TODO: Validate credentials for device %s, username %s against database\n", deviceID, username)
	return true // TEMPORARY: Replace with actual validation
}

func (s *Server) extractUUID(data []byte) string {
	// Parse the APF_PROTOCOL_VERSION_MESSAGE
	// Structure: MessageType (1 byte) + MajorVersion (4 bytes) + MinorVersion (4 bytes) +
	//            TriggerReason (4 bytes) + UUID (16 bytes) + Reserved (64 bytes)
	if len(data) >= 29 { // 1 + 4 + 4 + 4 + 16
		// Extract UUID starting at byte 13 (after MessageType + 3 uint32s)
		uuidBytes := data[13:29]

		// UUID format uses mixed endianness:
		// - First 4 bytes (Data1): little-endian
		// - Next 2 bytes (Data2): little-endian
		// - Next 2 bytes (Data3): little-endian
		// - Last 8 bytes (Data4): big-endian
		uuid := fmt.Sprintf("%02X%02X%02X%02X-%02X%02X-%02X%02X-%02X%02X-%02X%02X%02X%02X%02X%02X",
			uuidBytes[3], uuidBytes[2], uuidBytes[1], uuidBytes[0], // Data1 (reversed)
			uuidBytes[5], uuidBytes[4], // Data2 (reversed)
			uuidBytes[7], uuidBytes[6], // Data3 (reversed)
			uuidBytes[8], uuidBytes[9], // Data4 (not reversed)
			uuidBytes[10], uuidBytes[11], uuidBytes[12], uuidBytes[13], uuidBytes[14], uuidBytes[15])

		// Also parse other fields for logging
		reader := bytes.NewReader(data[1:13])
		var majorVersion, minorVersion, triggerReason uint32
		binary.Read(reader, binary.BigEndian, &majorVersion)
		binary.Read(reader, binary.BigEndian, &minorVersion)
		binary.Read(reader, binary.BigEndian, &triggerReason)

		log.Printf("APF Protocol Version Message detected - Version: %d.%d, Trigger: %d, UUID: %s\n",
			majorVersion, minorVersion, triggerReason, uuid)

		return uuid
	}
	return ""
}

// Shutdown -.
func (s *Server) Shutdown() error {
	if s.listener != nil {
		return s.listener.Close()
	}

	return nil
}
