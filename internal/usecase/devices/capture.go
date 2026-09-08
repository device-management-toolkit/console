package devices

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/client"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

// CaptureKVMOptions controls a one-shot KVM frame capture.
type CaptureKVMOptions struct {
	TimeoutSeconds int
	MaxWidth       int // hint for the caller/agent; scaling is not performed server-side
	MaxHeight      int
}

var (
	// ErrCaptureUnsupportedEncoding indicates the device sent an RFB encoding this
	// scaffold does not decode. Only RAW is advertised, so this normally means the
	// device ignored our SetEncodings request.
	ErrCaptureUnsupportedEncoding = errors.New("kvm capture: unsupported RFB encoding (scaffold advertises RAW only)")
	// ErrCaptureRedirectionStatus indicates the AMT redirection/RFB handshake was rejected.
	ErrCaptureRedirectionStatus = errors.New("kvm capture: redirection session was not accepted by the device")
	// ErrCaptureNoFrame indicates no framebuffer update was received.
	ErrCaptureNoFrame = errors.New("kvm capture: no framebuffer update received")
)

const (
	kvmCaptureDefaultTimeout = 20 * time.Second
	kvmStartHeaderLen        = 8
	rfbTileMaxDimension      = 64
	captureBytesPerPixel     = 1 // 8-bit RGB332 pixel format we request
	rfbFramebufferAttempts   = 3
)

// AMT redirection control messages (browser side of the handshake).
var (
	// startRedirectionKVM is the AMT "start redirection" command for a KVM session ("KVMR").
	startRedirectionKVM = []byte{0x10, 0x01, 0x00, 0x00, 0x4b, 0x56, 0x4d, 0x52}
	// authQuery asks Intel AMT which authentication methods it supports.
	authQuery = []byte{0x13, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	// startKVMRedirect switches the session into direct KVM traffic after auth.
	startKVMRedirect = []byte{0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

// RFB client messages.
var (
	// rfbClientVersion is the RFB protocol version this client speaks.
	rfbClientVersion = []byte("RFB 003.008\n")
	// rfbSetEncodingsRAW advertises RAW (0) + DesktopSize pseudo-encoding (-223) only.
	// Omitting ZRLE (16) keeps tiles uncompressed so the raw bytes are directly usable.
	rfbSetEncodingsRAW = []byte{0x02, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0xFF, 0x21}
	// rfbSetPixelFormatRGB332 requests an 8-bit RGB332 pixel format (1 byte per pixel).
	rfbSetPixelFormatRGB332 = []byte{0, 0, 0, 0, 8, 8, 0, 1, 0, 7, 0, 7, 0, 3, 5, 2, 0, 0, 0, 0}
)

// CaptureKVMFrame opens a headless KVM redirection session to the device, drives
// the AMT redirection + RFB handshake (reusing Console's digest-auth injection),
// captures a single full-screen framebuffer as raw RGB332 pixels, and returns it.
//
// This is a capture-pipeline scaffold: it advertises RAW encoding only and does
// not decode compressed (ZRLE) tiles or render an image. Turning the raw
// framebuffer into a viewable image is left to the caller (agentic AI / VLM).
func (uc *UseCase) CaptureKVMFrame(ctx context.Context, guid string, opts CaptureKVMOptions) (dto.KVMFrame, error) {
	device, err := uc.repo.GetByID(ctx, guid, "")
	if err != nil {
		return dto.KVMFrame{}, err
	}

	if device == nil || device.GUID == "" {
		return dto.KVMFrame{}, ErrNotFound
	}

	wsmanConnection, err := uc.redirection.SetupWsmanClient(ctx, *device, true, true)
	if err != nil {
		return dto.KVMFrame{}, err
	}

	decryptedPassword, err := uc.safeRequirements.Decrypt(device.Password)
	if err != nil {
		return dto.KVMFrame{}, err
	}

	timeout := kvmCaptureDefaultTimeout
	if opts.TimeoutSeconds > 0 {
		timeout = time.Duration(opts.TimeoutSeconds) * time.Second
	}

	sessionCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	deviceConnection := &DeviceConnection{
		wsmanMessages: wsmanConnection,
		Device:        *device,
		Mode:          "kvm",
		Challenge: client.AuthChallenge{
			Username: device.Username,
			Password: decryptedPassword,
		},
		ctx:    sessionCtx,
		cancel: cancel,
	}

	if err = uc.redirection.RedirectConnect(sessionCtx, deviceConnection); err != nil {
		return dto.KVMFrame{}, err
	}

	defer func() { _ = uc.redirection.RedirectClose(context.Background(), deviceConnection) }()

	stream := &redirectStream{uc: uc, dc: deviceConnection}

	if err = uc.performRedirectionHandshake(deviceConnection, stream); err != nil {
		return dto.KVMFrame{}, err
	}

	frame, err := uc.captureRFBFrame(stream)
	if err != nil {
		return dto.KVMFrame{}, err
	}

	frame.GUID = device.GUID
	frame.CapturedAt = time.Now().UTC()
	frame.RequestedMaxWidth = opts.MaxWidth
	frame.RequestedMaxHeight = opts.MaxHeight

	return frame, nil
}

// redirectStream is a small buffered reader/writer over the redirection channel.
type redirectStream struct {
	uc  *UseCase
	dc  *DeviceConnection
	buf []byte
}

func (s *redirectStream) send(data []byte) error {
	return s.uc.redirection.RedirectSend(s.dc.ctx, s.dc, data)
}

func (s *redirectStream) fill() error {
	data, err := s.uc.redirection.RedirectListen(s.dc.ctx, s.dc)
	if err != nil {
		return err
	}

	s.buf = append(s.buf, data...)

	return nil
}

// read returns exactly n bytes, blocking (until context timeout) for more data.
func (s *redirectStream) read(n int) ([]byte, error) {
	for len(s.buf) < n {
		if err := s.fill(); err != nil {
			return nil, err
		}
	}

	out := make([]byte, n)
	copy(out, s.buf[:n])
	s.buf = s.buf[n:]

	return out, nil
}

func (uc *UseCase) performRedirectionHandshake(deviceConnection *DeviceConnection, s *redirectStream) error {
	// 1. Start redirection session (KVM).
	if err := s.send(startRedirectionKVM); err != nil {
		return err
	}

	if err := readStartRedirectionReply(s); err != nil {
		return err
	}

	// 2. Query supported authentication methods.
	if err := s.send(authQuery); err != nil {
		return err
	}

	if _, err := readAuthReply(s, &deviceConnection.Challenge); err != nil {
		return err
	}

	// 3. Initial (empty) digest auth to obtain the realm/nonce challenge.
	if err := s.send(handleDigestAuthentication(&deviceConnection.Challenge)); err != nil {
		return err
	}

	direct, err := readAuthReply(s, &deviceConnection.Challenge)
	if err != nil {
		return err
	}

	// 4. Full digest auth (Console computes the digest from the challenge).
	if !direct {
		if err = s.send(handleDigestAuthentication(&deviceConnection.Challenge)); err != nil {
			return err
		}

		direct, err = readAuthReply(s, &deviceConnection.Challenge)
		if err != nil {
			return err
		}
	}

	if !direct {
		return ErrCaptureRedirectionStatus
	}

	// 5. Switch to direct KVM traffic and consume the 0x41 acknowledgement.
	if err = s.send(startKVMRedirect); err != nil {
		return err
	}

	return readKVMStart(s)
}

func readStartRedirectionReply(s *redirectStream) error {
	head, err := s.read(RedirectSessionLengthBytes)
	if err != nil {
		return err
	}

	if head[1] != StartRedirectionSessionReplyStatusSuccess {
		return ErrCaptureRedirectionStatus
	}

	oemLen := int(head[12])
	if oemLen > 0 {
		if _, err := s.read(oemLen); err != nil {
			return err
		}
	}

	return nil
}

// readAuthReply reads one AuthenticateSessionReply, updates the challenge with
// any realm/nonce it carries, and reports whether the device signalled that
// authentication succeeded.
func readAuthReply(s *redirectStream, challenge *client.AuthChallenge) (bool, error) {
	head, err := s.read(HeaderByteSize)
	if err != nil {
		return false, err
	}

	num := int(binary.LittleEndian.Uint32(head[5:HeaderByteSize]))

	full := head
	if num > 0 {
		rest, err := s.read(num)
		if err != nil {
			return false, err
		}

		full = append(full, rest...)
	}

	_, direct := handleAuthenticateSessionReply(full, challenge)

	return direct, nil
}

func readKVMStart(s *redirectStream) error {
	head, err := s.read(kvmStartHeaderLen)
	if err != nil {
		return err
	}

	if head[0] != 0x41 {
		return ErrCaptureRedirectionStatus
	}

	return nil
}

// captureRFBFrame runs the RFB (VNC) handshake and captures a single frame.
func (uc *UseCase) captureRFBFrame(s *redirectStream) (dto.KVMFrame, error) {
	// Server ProtocolVersion -> client ProtocolVersion.
	if _, err := s.read(len(rfbClientVersion)); err != nil {
		return dto.KVMFrame{}, err
	}

	if err := s.send(rfbClientVersion); err != nil {
		return dto.KVMFrame{}, err
	}

	// Security types: 1-byte count + that many type bytes. We already authenticated
	// via redirection digest auth, so we select "None" (type 1).
	count, err := s.read(1)
	if err != nil {
		return dto.KVMFrame{}, err
	}

	if count[0] > 0 {
		if _, err = s.read(int(count[0])); err != nil {
			return dto.KVMFrame{}, err
		}
	}

	if err = s.send([]byte{0x01}); err != nil {
		return dto.KVMFrame{}, err
	}

	// SecurityResult (0 = OK).
	res, err := s.read(4)
	if err != nil {
		return dto.KVMFrame{}, err
	}

	if binary.BigEndian.Uint32(res) != 0 {
		return dto.KVMFrame{}, ErrCaptureRedirectionStatus
	}

	// ClientInit: shared-desktop flag.
	if err = s.send([]byte{0x01}); err != nil {
		return dto.KVMFrame{}, err
	}

	// ServerInit: framebuffer width/height + pixel format + name.
	head, err := s.read(24)
	if err != nil {
		return dto.KVMFrame{}, err
	}

	width := int(binary.BigEndian.Uint16(head[0:2]))
	height := int(binary.BigEndian.Uint16(head[2:4]))

	nameLen := int(binary.BigEndian.Uint32(head[20:24]))
	if nameLen > 0 {
		if _, err = s.read(nameLen); err != nil {
			return dto.KVMFrame{}, err
		}
	}

	// Advertise RAW encoding + request an 8-bit RGB332 pixel format.
	if err = s.send(rfbSetEncodingsRAW); err != nil {
		return dto.KVMFrame{}, err
	}

	if err = s.send(rfbSetPixelFormatRGB332); err != nil {
		return dto.KVMFrame{}, err
	}

	return uc.readFramebufferUpdate(s, width, height)
}

// readFramebufferUpdate requests and assembles one full-screen framebuffer.
func (uc *UseCase) readFramebufferUpdate(s *redirectStream, width, height int) (dto.KVMFrame, error) {
	for attempt := 0; attempt < rfbFramebufferAttempts; attempt++ {
		if width <= 0 || height <= 0 {
			return dto.KVMFrame{}, ErrCaptureNoFrame
		}

		if err := s.send(framebufferUpdateRequest(0, 0, 0, width, height)); err != nil {
			return dto.KVMFrame{}, err
		}

		framebuffer := make([]byte, width*height*captureBytesPerPixel)

		newWidth, newHeight, resized, rectangles, err := readOneUpdate(s, framebuffer, width, height)
		if err != nil {
			return dto.KVMFrame{}, err
		}

		if resized {
			// DesktopSize pseudo-encoding: re-request against the new dimensions.
			width, height = newWidth, newHeight

			continue
		}

		return dto.KVMFrame{
			Width:          width,
			Height:         height,
			BytesPerPixel:  captureBytesPerPixel,
			PixelFormat:    "RGB332",
			Encoding:       "raw",
			RectangleCount: rectangles,
			DataBase64:     base64.StdEncoding.EncodeToString(framebuffer),
		}, nil
	}

	return dto.KVMFrame{}, ErrCaptureNoFrame
}

// framebufferUpdateRequest builds an RFB FramebufferUpdateRequest message.
func framebufferUpdateRequest(incremental byte, x, y, w, h int) []byte {
	msg := make([]byte, 10)
	msg[0] = 3
	msg[1] = incremental
	binary.BigEndian.PutUint16(msg[2:4], uint16(x))
	binary.BigEndian.PutUint16(msg[4:6], uint16(y))
	binary.BigEndian.PutUint16(msg[6:8], uint16(w))
	binary.BigEndian.PutUint16(msg[8:10], uint16(h))

	return msg
}

// awaitFramebufferUpdate consumes RFB server messages until a FramebufferUpdate
// (type 0) arrives, skipping Bell (2) and ServerCutText (3) messages.
func (s *redirectStream) awaitFramebufferUpdate() error {
	for {
		msgType, err := s.read(1)
		if err != nil {
			return err
		}

		switch msgType[0] {
		case 0:
			return nil
		case 2:
			continue
		case 3:
			hdr, err := s.read(7) // 3 padding + 4-byte length
			if err != nil {
				return err
			}

			length := int(binary.BigEndian.Uint32(hdr[3:7]))
			if length > 0 {
				if _, err = s.read(length); err != nil {
					return err
				}
			}
		default:
			return ErrCaptureUnsupportedEncoding
		}
	}
}

// readOneUpdate parses a single FramebufferUpdate and blits its RAW rectangles
// into framebuffer. It returns updated dimensions and resized=true if the update
// carried a DesktopSize pseudo-rectangle instead of pixel data.
func readOneUpdate(s *redirectStream, framebuffer []byte, width, height int) (int, int, bool, int, error) {
	if err := s.awaitFramebufferUpdate(); err != nil {
		return 0, 0, false, 0, err
	}

	head, err := s.read(3) // 1 padding byte + 2-byte rectangle count
	if err != nil {
		return 0, 0, false, 0, err
	}

	numRects := int(binary.BigEndian.Uint16(head[1:3]))

	for i := 0; i < numRects; i++ {
		rect, err := s.read(12)
		if err != nil {
			return 0, 0, false, 0, err
		}

		rx := int(binary.BigEndian.Uint16(rect[0:2]))
		ry := int(binary.BigEndian.Uint16(rect[2:4]))
		rw := int(binary.BigEndian.Uint16(rect[4:6]))
		rh := int(binary.BigEndian.Uint16(rect[6:8]))
		encoding := int32(binary.BigEndian.Uint32(rect[8:12]))

		switch encoding {
		case 0: // RAW
			if rw < 1 || rw > rfbTileMaxDimension || rh < 1 || rh > rfbTileMaxDimension {
				return 0, 0, false, 0, fmt.Errorf("%w: tile %dx%d", ErrCaptureUnsupportedEncoding, rw, rh)
			}

			data, err := s.read(rw * rh * captureBytesPerPixel)
			if err != nil {
				return 0, 0, false, 0, err
			}

			blitRAW(framebuffer, width, height, rx, ry, rw, rh, data)
		case -223: // DesktopSize pseudo-encoding carries the new screen size in rw/rh.
			return rw, rh, true, i, nil
		default:
			return 0, 0, false, 0, ErrCaptureUnsupportedEncoding
		}
	}

	return width, height, false, numRects, nil
}

// blitRAW copies an 8-bit RAW tile into the full framebuffer at (rx, ry).
func blitRAW(framebuffer []byte, width, height, rx, ry, rw, rh int, data []byte) {
	for row := 0; row < rh; row++ {
		dy := ry + row
		if dy < 0 || dy >= height {
			continue
		}

		for col := 0; col < rw; col++ {
			dx := rx + col
			if dx < 0 || dx >= width {
				continue
			}

			framebuffer[dy*width+dx] = data[row*rw+col]
		}
	}
}
