//nolint:mnd // APF protocol message sizes are defined by the specification
package cira

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/apf"
	wsman2 "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/client"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const (
	maxIdleTime          = 300 * time.Second
	port                 = "4433"
	readBufferSize       = 4096
	weakCipherSuiteCount = 3
	keepAliveInterval    = 30
	keepAliveTimeout     = 90
	apfSessionTimeout    = 3 * time.Second
)

var (
	mu sync.Mutex

	// ErrChannelOpenFailed is returned when an APF channel open request fails.
	ErrChannelOpenFailed = errors.New("channel open failed")
)

type Server struct {
	certificates tls.Certificate
	notify       chan error
	listener     net.Listener
	devices      devices.Feature
	log          logger.Interface
}

func NewServer(certFile, keyFile string, d devices.Feature, l logger.Interface) (*Server, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	s := &Server{
		certificates: cert,
		notify:       make(chan error, 1),
		devices:      d,
		log:          l,
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
		Certificates: []tls.Certificate{s.certificates},
		// InsecureSkipVerify is set to true because this is a TLS server accepting
		// client connections from AMT devices. The server does not need to verify
		// its own certificate. Client authentication is handled at the APF protocol level.
		InsecureSkipVerify: true, //nolint:gosec // Server-side TLS config, not a client connection
		MinVersion:         tls.VersionTLS12,
	}

	defaultCipherSuites := tls.CipherSuites()
	config.CipherSuites = make([]uint16, 0, len(defaultCipherSuites)+weakCipherSuiteCount)

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

	s.log.Info("CIRA server running on port %s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go s.handleConnection(conn)
	}
}

type connectionContext struct {
	conn          net.Conn
	tlsConn       *tls.Conn
	handler       *APFHandler
	processor     *apf.Processor
	session       *apf.Session
	authenticated bool
	device        *wsman.ConnectionEntry
	log           logger.Interface
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		s.log.Error("Failed to cast connection to TLS connection")

		return
	}

	s.log.Debug("New TLS connection from %s", conn.RemoteAddr())

	ctx := &connectionContext{
		conn:    conn,
		tlsConn: tlsConn,
		handler: NewAPFHandler(s.devices, s.log),
		session: &apf.Session{
			Timer: time.NewTimer(apfSessionTimeout),
		},
		log: s.log,
	}
	ctx.processor = apf.NewProcessor(ctx.handler)

	defer ctx.cleanup()

	s.processConnection(ctx)
}

func (ctx *connectionContext) cleanup() {
	deviceID := ctx.handler.DeviceID()
	if ctx.authenticated && deviceID != "" {
		mu.Lock()
		delete(wsman.Connections, deviceID)
		mu.Unlock()
	}

	// Stop and clean up the session timer
	if ctx.session != nil && ctx.session.Timer != nil {
		if !ctx.session.Timer.Stop() {
			// Drain the channel if the timer fired
			select {
			case <-ctx.session.Timer.C:
			default:
			}
		}
	}
}

func (s *Server) processConnection(ctx *connectionContext) {
	for {
		if shouldReturn := ctx.processNextMessage(); shouldReturn {
			return
		}
	}
}

// processNextMessage handles the next message from the connection.
// Returns true if the connection should be closed.
func (ctx *connectionContext) processNextMessage() (shouldReturn bool) {
	if err := ctx.conn.SetDeadline(time.Now().Add(maxIdleTime)); err != nil {
		ctx.log.Error("Failed to set deadline: %v", err)

		return true
	}

	data, err := ctx.readData()
	if err != nil {
		return true
	}

	messageType := ctx.getMessageType(data)

	// Route APF channel messages to registered channels
	if ctx.authenticated && ctx.device != nil && ctx.handleAPFChannelMessage(messageType, data) {
		return false // Message was handled by channel routing, continue loop
	}

	response := ctx.processor.Process(data, ctx.session)

	if !ctx.handleAuthFlow(messageType, response) {
		return true
	}

	if err := ctx.writeResponse(response); err != nil {
		return true
	}

	if err := ctx.sendKeepAliveIfNeeded(messageType); err != nil {
		return true
	}

	return false
}

func (ctx *connectionContext) readData() ([]byte, error) {
	buf := make([]byte, readBufferSize)

	n, err := ctx.tlsConn.Read(buf)
	if err != nil && n == 0 {
		deviceID := ctx.handler.DeviceID()
		if errors.Is(err, net.ErrClosed) {
			ctx.log.Info("Connection closed for device %s", deviceID)
		} else {
			ctx.log.Warn("Read error for device %s: %v", deviceID, err)
		}

		return nil, err
	}

	data := buf[:n]
	ctx.log.Debug("Received data from %s: %s", ctx.handler.DeviceID(), hex.EncodeToString(data))

	return data, nil
}

func (ctx *connectionContext) getMessageType(data []byte) byte {
	if len(data) > 0 {
		return data[0]
	}

	return 0
}

func (ctx *connectionContext) handleAuthFlow(messageType byte, response bytes.Buffer) bool {
	if messageType != apf.APF_USERAUTH_REQUEST || ctx.authenticated {
		return true
	}

	responseBytes := response.Bytes()
	if len(responseBytes) > 0 && responseBytes[0] == apf.APF_USERAUTH_SUCCESS {
		ctx.registerDevice()

		return true
	}

	// Authentication failed - send response and close connection
	_, _ = ctx.conn.Write(responseBytes)

	ctx.log.Warn("Authentication failed for device, closing connection")

	return false
}

func (ctx *connectionContext) registerDevice() {
	ctx.authenticated = true
	deviceID := ctx.handler.DeviceID()

	ctx.device = &wsman.ConnectionEntry{
		IsCIRA:        true,
		Conny:         ctx.conn,
		Timer:         time.NewTimer(maxIdleTime),
		WsmanMessages: wsman2.NewMessages(client.Parameters{}),
	}

	mu.Lock()

	wsman.Connections[deviceID] = ctx.device

	mu.Unlock()

	ctx.log.Info("Device authenticated and registered: %s", deviceID)
}

func (ctx *connectionContext) writeResponse(response bytes.Buffer) error {
	if _, err := ctx.conn.Write(response.Bytes()); err != nil {
		ctx.log.Error("Write error for device %s: %v", ctx.handler.DeviceID(), err)

		return err
	}

	return nil
}

func (ctx *connectionContext) sendKeepAliveIfNeeded(messageType byte) error {
	if !ctx.authenticated || messageType != apf.APF_GLOBAL_REQUEST || !ctx.handler.ShouldSendKeepAlive() {
		return nil
	}

	var binBuf bytes.Buffer

	keepAliveOptionsRequest := apf.KeepAliveOptionsRequest(keepAliveInterval, keepAliveTimeout)

	if err := binary.Write(&binBuf, binary.BigEndian, keepAliveOptionsRequest); err != nil {
		ctx.log.Error("Error creating keep-alive request: %v", err)

		return nil // Continue processing, don't break connection
	}

	if _, err := ctx.conn.Write(binBuf.Bytes()); err != nil {
		ctx.log.Error("Error sending keep-alive request: %v", err)

		return err
	}

	ctx.log.Debug("Sent keep-alive options request for device %s", ctx.handler.DeviceID())

	return nil
}

// handleAPFChannelMessage routes APF channel messages to registered channels.
// Returns true if the message was handled by channel routing.
func (ctx *connectionContext) handleAPFChannelMessage(messageType byte, data []byte) bool {
	switch messageType {
	case apf.APF_CHANNEL_OPEN_CONFIRMATION:
		return ctx.handleChannelOpenConfirmation(data)
	case apf.APF_CHANNEL_OPEN_FAILURE:
		return ctx.handleChannelOpenFailure(data)
	case apf.APF_CHANNEL_DATA:
		return ctx.handleChannelData(data)
	case apf.APF_CHANNEL_WINDOW_ADJUST:
		return ctx.handleChannelWindowAdjust(data)
	case apf.APF_CHANNEL_CLOSE:
		return ctx.handleChannelClose(data)
	default:
		return false
	}
}

func (ctx *connectionContext) handleChannelOpenConfirmation(data []byte) bool {
	if len(data) < 17 {
		return false
	}

	// Parse: [type(1)][recipient(4)][sender(4)][window(4)][reserved(4)]
	recipientChannel := binary.BigEndian.Uint32(data[1:5])
	senderChannel := binary.BigEndian.Uint32(data[5:9])
	initialWindow := binary.BigEndian.Uint32(data[9:13])

	if ctx.device == nil {
		return false
	}

	channel := ctx.device.GetAPFChannel(recipientChannel)
	if channel == nil {
		ctx.log.Debug("No registered channel for recipient %d", recipientChannel)

		return false
	}

	// Use thread-safe setters
	channel.SetRecipientChannel(senderChannel)
	channel.SetTXWindow(initialWindow)

	// Signal success
	channel.SignalOpen(nil)

	ctx.log.Debug("APF channel opened: our=%d, device=%d, window=%d",
		channel.GetSenderChannel(), channel.GetRecipientChannel(), channel.GetTXWindow())

	return true
}

func (ctx *connectionContext) handleChannelOpenFailure(data []byte) bool {
	if len(data) < 9 {
		return false
	}

	recipientChannel := binary.BigEndian.Uint32(data[1:5])
	reasonCode := binary.BigEndian.Uint32(data[5:9])

	channel := ctx.device.GetAPFChannel(recipientChannel)
	if channel == nil {
		return false
	}

	// Signal failure
	channel.SignalOpen(fmt.Errorf("%w: reason code %d", ErrChannelOpenFailed, reasonCode))

	ctx.device.UnregisterAPFChannel(recipientChannel)

	return true
}

func (ctx *connectionContext) handleChannelData(data []byte) bool {
	if len(data) < 9 {
		return false
	}

	// recipientChannel in the message is OUR sender channel ID
	ourChannel := binary.BigEndian.Uint32(data[1:5])
	dataLen := binary.BigEndian.Uint32(data[5:9])

	if len(data) < int(9+dataLen) {
		return false
	}

	if ctx.device == nil {
		return false
	}

	channelData := data[9 : 9+dataLen]

	// Look up by our sender channel ID (not by recipient)
	channel := ctx.device.GetAPFChannel(ourChannel)
	if channel == nil {
		return false
	}

	// Send data to the channel
	channel.SendData(channelData)

	return true
}

func (ctx *connectionContext) handleChannelWindowAdjust(data []byte) bool {
	if len(data) < 9 {
		return false
	}

	// recipientChannel in the message is OUR sender channel ID
	ourChannel := binary.BigEndian.Uint32(data[1:5])
	bytesToAdd := binary.BigEndian.Uint32(data[5:9])

	if ctx.device == nil {
		return false
	}

	// Look up by our sender channel ID (not by recipient)
	channel := ctx.device.GetAPFChannel(ourChannel)
	if channel == nil {
		return false
	}

	// Send window adjust to the channel
	channel.SendWindowAdjust(bytesToAdd)

	return true
}

func (ctx *connectionContext) handleChannelClose(data []byte) bool {
	if len(data) < 5 {
		return false
	}

	// recipientChannel in the message is OUR sender channel ID
	ourChannel := binary.BigEndian.Uint32(data[1:5])

	// Look up by our sender channel ID (not by recipient)
	channel := ctx.device.GetAPFChannel(ourChannel)
	if channel == nil {
		return false
	}

	ctx.device.UnregisterAPFChannel(ourChannel)

	return true
}

// Shutdown gracefully shuts down the CIRA server.
func (s *Server) Shutdown() error {
	if s.listener != nil {
		return s.listener.Close()
	}

	return nil
}
