package devices

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"sync"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/client"
	"github.com/gorilla/websocket"

	"github.com/device-management-toolkit/console/internal/entity"
)

const (
	HeaderByteSize             = 9
	ContentLengthPadding       = 8
	RedirectSessionLengthBytes = 13
	RedirectionSessionReply    = 4
	ConnectionTimeout          = 5 * time.Minute
	InactivityTimeout          = 30 * time.Second // Close connection if no data for 30 seconds
	HeartbeatInterval          = 30 * time.Second // Check connection health every 30 seconds
)

type DeviceConnection struct {
	Conn          WebSocketConn
	wsmanMessages wsman.Messages
	Device        entity.Device
	Direct        bool
	Mode          string
	Challenge     client.AuthChallenge
	ctx           context.Context
	cancel        context.CancelFunc
	lastActivity  time.Time
	lastDataRecv  time.Time // Track last data received from device
	mu            sync.RWMutex
	healthTicker  *time.Ticker
}

func (uc *UseCase) Redirect(c context.Context, conn *websocket.Conn, guid, mode string) error {
	device, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return err
	}

	if device == nil || device.GUID == "" {
		return ErrNotFound
	}

	key := device.GUID + "-" + mode

	deviceConnection, err := uc.getOrCreateConnection(c, conn, key, device)
	if err != nil {
		return err
	}

	err = uc.redirection.RedirectConnect(c, deviceConnection)
	if err != nil {
		deviceConnection.cancel()

		uc.redirMutex.Lock()
		delete(uc.redirConnections, key)
		uc.redirMutex.Unlock()

		return err
	}

	uc.updateConnectionActivity(deviceConnection)
	uc.startConnectionGoroutines(c, deviceConnection, key)

	return nil
}

func (uc *UseCase) getOrCreateConnection(c context.Context, conn *websocket.Conn, key string, device *entity.Device) (*DeviceConnection, error) {
	uc.redirMutex.RLock()
	existingConn, ok := uc.redirConnections[key]
	uc.redirMutex.RUnlock()

	if ok {
		// Check if existing connection is still valid
		existingConn.mu.RLock()
		isExpired := time.Since(existingConn.lastActivity) > ConnectionTimeout
		existingConn.mu.RUnlock()

		if isExpired {
			// Clean up expired connection
			existingConn.cancel()
			uc.redirection.RedirectClose(c, existingConn)

			uc.redirMutex.Lock()
			delete(uc.redirConnections, key)
			uc.redirMutex.Unlock()
		} else {
			existingConn.Conn = conn // Update websocket connection

			return existingConn, nil
		}
	}

	return uc.createNewConnection(c, conn, key, device)
}

func (uc *UseCase) createNewConnection(c context.Context, conn *websocket.Conn, key string, device *entity.Device) (*DeviceConnection, error) {
	wsmanConnection := uc.redirection.SetupWsmanClient(*device, true, true)

	device.Password, _ = uc.safeRequirements.Decrypt(device.Password)

	ctx, cancel := context.WithCancel(c)
	now := time.Now()
	deviceConnection := &DeviceConnection{
		Conn:          conn,
		wsmanMessages: wsmanConnection,
		Device:        *device,
		Direct:        false,
		Mode:          key[len(device.GUID)+1:], // Extract mode from key
		Challenge: client.AuthChallenge{
			Username: device.Username,
			Password: device.Password,
		},
		ctx:          ctx,
		cancel:       cancel,
		lastActivity: now,
		lastDataRecv: now,
		healthTicker: time.NewTicker(HeartbeatInterval),
	}

	uc.redirMutex.Lock()
	uc.redirConnections[key] = deviceConnection
	uc.redirMutex.Unlock()

	return deviceConnection, nil
}

func (uc *UseCase) updateConnectionActivity(deviceConnection *DeviceConnection) {
	deviceConnection.mu.Lock()
	deviceConnection.lastActivity = time.Now()
	deviceConnection.mu.Unlock()
}

func (uc *UseCase) startConnectionGoroutines(c context.Context, deviceConnection *DeviceConnection, key string) {
	var wg sync.WaitGroup

	const numGoroutines = 3 // Device listener, Browser listener, Health monitor

	wg.Add(numGoroutines)

	go func() {
		defer wg.Done()
		uc.ListenToDevice(deviceConnection)
	}()

	go func() {
		defer wg.Done()
		uc.ListenToBrowser(deviceConnection)
	}()

	go func() {
		defer wg.Done()
		uc.MonitorConnectionHealth(deviceConnection, key)
	}()

	// Start cleanup goroutine
	go func() {
		wg.Wait()
		// All goroutines finished, clean up
		if deviceConnection.healthTicker != nil {
			deviceConnection.healthTicker.Stop()
		}

		deviceConnection.cancel()
		uc.redirection.RedirectClose(c, deviceConnection)

		uc.redirMutex.Lock()
		delete(uc.redirConnections, key)
		uc.redirMutex.Unlock()
	}()
}

func (uc *UseCase) ListenToDevice(deviceConnection *DeviceConnection) {
	conn := deviceConnection.Conn

	defer func() {
		// Clean up on exit
		deviceConnection.cancel()
	}()

	for {
		select {
		case <-deviceConnection.ctx.Done():
			return
		default:
		}

		// Update last activity time
		deviceConnection.mu.Lock()
		deviceConnection.lastActivity = time.Now()
		deviceConnection.mu.Unlock()

		// Measure time blocked waiting for device data
		recvStart := time.Now()
		data, err := uc.redirection.RedirectListen(deviceConnection.ctx, deviceConnection)
		kvmDeviceReceiveBlockSeconds.WithLabelValues(deviceConnection.Mode).Observe(time.Since(recvStart).Seconds())

		if err != nil {
			break
		}

		if len(data) == 0 {
			continue
		}

		// Update last data received timestamp
		deviceConnection.mu.Lock()
		deviceConnection.lastDataRecv = time.Now()
		deviceConnection.mu.Unlock()

		toSend := data
		if !deviceConnection.Direct {
			toSend, deviceConnection.Direct = processDeviceData(toSend, &deviceConnection.Challenge)
		}

		// metrics: device -> browser
		start := time.Now()

		kvmDevicePayloadBytes.WithLabelValues(deviceConnection.Mode).Observe(float64(len(toSend)))
		kvmDeviceToBrowserBytes.WithLabelValues(deviceConnection.Mode).Add(float64(len(toSend)))
		kvmDeviceToBrowserMessages.WithLabelValues(deviceConnection.Mode).Inc()

		err = conn.WriteMessage(websocket.BinaryMessage, toSend)

		kvmDeviceToBrowserWriteSeconds.WithLabelValues(deviceConnection.Mode).Observe(time.Since(start).Seconds())

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				_ = fmt.Errorf("interceptor - listenToDevice - websocket closed unexpectedly (writing to browser): %w", err)
			}

			return
		}
	}
}

func (uc *UseCase) ListenToBrowser(deviceConnection *DeviceConnection) {
	defer func() {
		// Clean up on exit
		deviceConnection.cancel()
	}()

	for {
		select {
		case <-deviceConnection.ctx.Done():
			return
		default:
		}

		// Update last activity time
		deviceConnection.mu.Lock()
		deviceConnection.lastActivity = time.Now()
		deviceConnection.mu.Unlock()

		readStart := time.Now()
		_, msg, err := deviceConnection.Conn.ReadMessage()
		kvmBrowserReadBlockSeconds.WithLabelValues(deviceConnection.Mode).Observe(time.Since(readStart).Seconds())

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				_ = fmt.Errorf("interceptor - listenToBrowser - websocket closed unexpectedly (reading from browser): %w", err)
			}

			return
		}

		toSend := msg
		if !deviceConnection.Direct {
			toSend = processBrowserData(msg, &deviceConnection.Challenge)
		}

		if len(toSend) == 0 {
			continue
		}

		// metrics: browser -> device
		start := time.Now()

		kvmBrowserPayloadBytes.WithLabelValues(deviceConnection.Mode).Observe(float64(len(toSend)))
		kvmBrowserToDeviceBytes.WithLabelValues(deviceConnection.Mode).Add(float64(len(toSend)))
		kvmBrowserToDeviceMessages.WithLabelValues(deviceConnection.Mode).Inc()
		// Send the message to the TCP Connection on the device
		err = uc.redirection.RedirectSend(deviceConnection.ctx, deviceConnection, toSend)
		kvmBrowserToDeviceSendSeconds.WithLabelValues(deviceConnection.Mode).Observe(time.Since(start).Seconds())

		if err != nil {
			_ = fmt.Errorf("interceptor - listenToBrowser - error sending message to device: %w", err)

			return
		}
	}
}

func (uc *UseCase) MonitorConnectionHealth(deviceConnection *DeviceConnection, key string) {
	defer func() {
		// Clean up on exit
		deviceConnection.cancel()
	}()

	for {
		select {
		case <-deviceConnection.ctx.Done():
			return
		case <-deviceConnection.healthTicker.C:
			deviceConnection.mu.RLock()
			lastDataTime := deviceConnection.lastDataRecv
			deviceConnection.mu.RUnlock()

			// Check if device has been inactive for too long
			if time.Since(lastDataTime) > InactivityTimeout {
				// Device appears unresponsive, force close connection
				deviceConnection.cancel()

				uc.redirMutex.Lock()
				delete(uc.redirConnections, key)
				uc.redirMutex.Unlock()

				return
			}
		}
	}
}

func processBrowserData(msg []byte, challenge *client.AuthChallenge) []byte {
	switch msg[0] {
	case RedirectionCommandsStartRedirectionSession:
		return msg[0:8]
	case RedirectionCommandsEndRedirectionSession:
		return msg[0:4]
	case RedirectionCommandsAuthenticateSession:
		return handleAuthenticationSession(msg, challenge)
	default:
	}

	return nil
}

func processDeviceData(msg []byte, challenge *client.AuthChallenge) ([]byte, bool) {
	switch msg[0] {
	case RedirectionCommandsStartRedirectionSessionReply:
		return handleStartRedirectionSessionReply(msg), false
	case RedirectionCommandsAuthenticateSessionReply:
		return handleAuthenticateSessionReply(msg, challenge)
	default:
	}

	return nil, false
}

func handleStartRedirectionSessionReply(msg []byte) []byte {
	if len(msg) < RedirectionSessionReply {
		return []byte("")
	}

	if msg[1:2][0] == uint8(0) {
		if len(msg) < RedirectSessionLengthBytes {
			return []byte("")
		}

		oemLen := int(msg[12:13][0])
		if len(msg) < RedirectSessionLengthBytes+oemLen {
			return []byte("")
		}

		r := msg[0 : RedirectSessionLengthBytes+oemLen]

		return r
	}

	return []byte("")
}

func allZero(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}

	return true
}

func handleAuthenticateSessionReply(msg []byte, challenge *client.AuthChallenge) ([]byte, bool) {
	if len(msg) < HeaderByteSize {
		return []byte(""), false
	}

	buf := bytes.NewReader(msg[1:HeaderByteSize])

	var authStatus, authType uint8

	var unknown uint16

	var num uint32

	_ = binary.Read(buf, binary.LittleEndian, &authStatus)
	_ = binary.Read(buf, binary.LittleEndian, &unknown)
	_ = binary.Read(buf, binary.LittleEndian, &authType)
	_ = binary.Read(buf, binary.LittleEndian, &num)

	if len(msg) < HeaderByteSize+int(num) {
		return []byte(""), false
	}

	if authType == AuthenticationTypeDigest && authStatus == AuthenticationStatusFail {
		var realmLength, nonceLength, qopLength uint8

		buf2 := bytes.NewReader(msg[9:])

		_ = binary.Read(buf2, binary.LittleEndian, &realmLength)

		realm := make([]byte, realmLength)
		_ = binary.Read(buf2, binary.LittleEndian, &realm)
		_ = binary.Read(buf2, binary.LittleEndian, &nonceLength)

		nonce := make([]byte, nonceLength)
		_ = binary.Read(buf2, binary.LittleEndian, &nonce)

		_ = binary.Read(buf2, binary.LittleEndian, &qopLength)

		qop := make([]byte, qopLength)
		_ = binary.Read(buf2, binary.LittleEndian, &qop)

		challenge.Realm = string(realm)
		challenge.Nonce = string(nonce)
		challenge.Qop = string(qop)
	} else if authType != AuthenticationTypeQuery && authStatus == AuthenticationStatusSuccess {
		// Intel AMT relayed that authentication was successful, go to direct relay mode in both directions.
		return msg, true
	}

	return msg, false
}

func RandomValueHex(length int) (string, error) {
	divideByHalf := 2
	n := (length + 1) / divideByHalf // Calculate the number of bytes needed

	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err // Return the error if random byte generation fails
	}

	hexStr := hex.EncodeToString(b) // Convert bytes to a hexadecimal string

	return hexStr[:length], nil // Slice the string to the desired length and return it
}

// Helper function to write length and bytes.
func writeField(buf io.Writer, field string) error {
	// Check for potential overflow
	var fieldLen uint8
	if len(field) <= math.MaxUint8 {
		fieldLen = uint8(len(field)) //nolint:gosec // Ignore potential overflow here as overflow validated earlier in code
	} else {
		return ErrLengthLimit
	}

	if err := binary.Write(buf, binary.BigEndian, fieldLen); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.BigEndian, []byte(field)); err != nil {
		return err
	}

	return nil
}

func handleAuthenticationSession(msg []byte, challenge *client.AuthChallenge) []byte {
	if len(msg) < HeaderByteSize {
		return []byte("")
	}

	if len(msg) == 9 && allZero(msg[1:]) {
		return msg
	}

	return processAuthChallenge(msg[1:9], challenge)
}

func processAuthChallenge(data []byte, challenge *client.AuthChallenge) []byte {
	buf := bytes.NewReader(data)

	var status uint8

	var unknown uint16

	var authType uint8

	if err := readBinaryData(buf, &status, &unknown, &authType); err != nil {
		log.Printf("Error reading binary data: %v", err)

		return nil
	}

	if authType == AuthenticationTypeDigest {
		return handleDigestAuthentication(challenge)
	}

	return []byte("")
}

func readBinaryData(buf *bytes.Reader, status *uint8, unknown *uint16, authType *uint8) error {
	if err := binary.Read(buf, binary.BigEndian, status); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.BigEndian, unknown); err != nil {
		return err
	}

	return binary.Read(buf, binary.BigEndian, authType)
}

func handleDigestAuthentication(challenge *client.AuthChallenge) []byte {
	if challenge.Realm != "" {
		cnonce, err := generateCNonce(challenge)
		if err != nil {
			log.Printf("Error generating CNonce: %v", err)

			return nil
		}

		challenge.CNonce = cnonce
		response := computeDigestResponse(challenge)

		return buildAuthReply(challenge, response)
	}

	return generateEmptyAuth(challenge, "/RedirectionService")
}

func generateCNonce(challenge *client.AuthChallenge) (string, error) {
	randomByteCount := 10
	cnonce, err := RandomValueHex(randomByteCount)
	if err != nil { //nolint:wsl // ignoring cuddle assignment rule for this line due to linter conflicts
		return "", err
	}

	challenge.NonceCount++

	return cnonce, nil
}

func computeDigestResponse(challenge *client.AuthChallenge) string {
	nonceData := challenge.GetFormattedNonceData(challenge.Nonce)

	return challenge.ComputeDigestHash("POST", "/RedirectionService", nonceData)
}

func buildAuthReply(challenge *client.AuthChallenge, response string) []byte {
	var replyBuf bytes.Buffer

	if err := writeHeader(&replyBuf); err != nil {
		log.Printf("Error writing header: %v", err)

		return nil
	}

	if err := writeLength(&replyBuf, challenge, response); err != nil {
		log.Printf("Error writing length: %v", err)

		return nil
	}

	if err := writeFields(&replyBuf, challenge, response); err != nil {
		log.Printf("Error writing fields: %v", err)

		return nil
	}

	return replyBuf.Bytes()
}

func writeHeader(buf *bytes.Buffer) error {
	return binary.Write(buf, binary.BigEndian, [5]byte{0x13, 0x00, 0x00, 0x00, 0x04})
}

var ErrLengthLimit = errors.New("calculated length exceeds uint32 limit")

func writeLength(buf *bytes.Buffer, challenge *client.AuthChallenge, response string) error {
	totalLength := len(challenge.Username) + len(challenge.Realm) + len(challenge.Nonce) + len("/RedirectionService") +
		len(challenge.CNonce) + len(fmt.Sprintf("%08x", challenge.NonceCount)) + len(response) + len(challenge.Qop) +
		ContentLengthPadding

	if totalLength > math.MaxUint32 {
		return ErrLengthLimit // If total length is too large, throws an error and stops here
	}

	length := uint32(totalLength) //nolint:gosec // Ignore potential integer overflow here as overflow is validated earlier in code

	return binary.Write(buf, binary.LittleEndian, length)
}

func writeFields(buf *bytes.Buffer, challenge *client.AuthChallenge, response string) error {
	if err := writeField(buf, challenge.Username); err != nil {
		return err
	}

	if err := writeField(buf, challenge.Realm); err != nil {
		return err
	}

	if err := writeField(buf, challenge.Nonce); err != nil {
		return err
	}

	if err := writeField(buf, "/RedirectionService"); err != nil {
		return err
	}

	if err := writeField(buf, challenge.CNonce); err != nil {
		return err
	}

	if err := writeField(buf, fmt.Sprintf("%08x", challenge.NonceCount)); err != nil {
		return err
	}

	if err := writeField(buf, response); err != nil {
		return err
	}

	return writeField(buf, challenge.Qop)
}

func generateEmptyAuth(challenge *client.AuthChallenge, authURL string) []byte {
	var buf bytes.Buffer

	lenChallengeUsername := uint8(0)
	lenAuthURL := uint8(0)

	// If challenge has values that will cause overflow, stop them here
	lenChallengeUsername = uint8(len(challenge.Username)) //nolint:gosec // Ignore potential integer overflow here as overflow is being validated
	lenAuthURL = uint8(len(authURL))                      //nolint:gosec // Ignore potential integer overflow here as overflow is being validated

	emptyAuth := emptyAuth{
		usernameLength: lenChallengeUsername, // Use calculated safe value
		authURLPadding: [2]byte{0x00, 0x00},
		authURLLength:  lenAuthURL, // Use calculated safe value
		endPadding:     [4]byte{0x00, 0x00, 0x00, 0x00},
	}

	copy(emptyAuth.username[:], challenge.Username)
	copy(emptyAuth.authURL[:], authURL)

	_ = binary.Write(&buf, binary.BigEndian, [5]byte{0x13, 0x00, 0x00, 0x00, 0x04})                           // header
	_ = binary.Write(&buf, binary.LittleEndian, uint32(lenChallengeUsername+lenAuthURL)+ContentLengthPadding) // flip flop endian for content length
	_ = binary.Write(&buf, binary.BigEndian, emptyAuth)

	return buf.Bytes()
}

type emptyAuth struct {
	usernameLength uint8
	username       [5]byte
	authURLPadding [2]byte
	authURLLength  uint8
	authURL        [19]byte
	endPadding     [4]byte
}
