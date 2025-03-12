package cira

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/open-amt-cloud-toolkit/console/internal/usecase/devices"
	"github.com/open-amt-cloud-toolkit/console/internal/usecase/devices/wsman"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/apf"
	wsman2 "github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/client"
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
		Certificates:       []tls.Certificate{s.certificates},
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}

	defaultCipherSuites := tls.CipherSuites()
	config.CipherSuites = make([]uint16, 0, len(defaultCipherSuites)+3)

	for _, suite := range defaultCipherSuites {
		config.CipherSuites = append(config.CipherSuites, suite.ID)
	}
	// add the weak cipher suites
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

	// Initialize a new ConnectedDevice and handle the connection
	deviceID := generateDeviceID()
	clientParams := client.Parameters{}
	device := &wsman.ConnectionEntry{
		IsCIRA:        true,
		Conny:         conn,
		Timer:         time.NewTimer(maxIdleTime),
		WsmanMessages: wsman2.NewMessages(clientParams),
	}

	session := apf.Session{}

	mu.Lock()
	wsman.Connections[deviceID] = device
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(wsman.Connections, deviceID)
		mu.Unlock()
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

		err = s.processData(tlsConn, data, session)
		if err != nil {
			log.Printf("Data processing error for device %s: %v\n", deviceID, err)

			break
		}
	}
}

func (s *Server) processData(conn net.Conn, data []byte, session apf.Session) error {
	// Implement data processing logic here
	log.Printf("Received data: %s\n", hex.EncodeToString(data))

	idk := apf.Process(data, &session)

	_, err := conn.Write(idk.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func generateDeviceID() string {
	data := make([]byte, 16)

	_, err := rand.Read(data)
	if err != nil {
		log.Fatal(err)
	}

	return hex.EncodeToString(data)
}

// Shutdown -.
func (s *Server) Shutdown() error {
	if s.listener != nil {
		return s.listener.Close()
	}

	return nil
}
