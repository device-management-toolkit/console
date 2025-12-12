package cira

import (
	"context"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/apf"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// globalRequestThreshold is the number of global requests required before sending keep-alive.
const globalRequestThreshold = 4

// APFHandler implements apf.Handler for the CIRA server.
// It provides application-specific logic for authentication and device registration.
type APFHandler struct {
	devices            devices.Feature
	deviceID           string
	globalRequestCount int
	log                logger.Interface
}

// NewAPFHandler creates a new APF handler with access to the devices feature.
func NewAPFHandler(d devices.Feature, l logger.Interface) *APFHandler {
	return &APFHandler{
		devices: d,
		log:     l,
	}
}

// DeviceID returns the device ID extracted from the protocol version message.
func (h *APFHandler) DeviceID() string {
	return h.deviceID
}

// OnProtocolVersion is called when an APF_PROTOCOLVERSION message is received.
// Extracts and stores the device UUID for later use.
func (h *APFHandler) OnProtocolVersion(info apf.ProtocolVersionInfo) error {
	h.deviceID = info.UUID

	h.log.Debug("APF Protocol Version - Version: %d.%d, Trigger: %d, UUID: %s",
		info.MajorVersion, info.MinorVersion, info.TriggerReason, info.UUID)

	return nil
}

// OnAuthRequest is called when an APF_USERAUTH_REQUEST message is received.
// Validates credentials against the database.
func (h *APFHandler) OnAuthRequest(request apf.AuthRequest) apf.AuthResponse {
	h.log.Debug("Authentication attempt - Device: %s, Username: %s, Method: %s",
		h.deviceID, request.Username, request.MethodName)

	// Only support password authentication
	if request.MethodName != "password" {
		h.log.Warn("Unsupported authentication method: %s", request.MethodName)

		return apf.AuthResponse{Authenticated: false}
	}

	// Validate credentials against database
	isValid := h.validateCredentials(request.Username, request.Password)

	if isValid {
		h.log.Debug("Authentication successful for device %s", h.deviceID)
	} else {
		h.log.Warn("Authentication failed for device %s with username %s",
			h.deviceID, request.Username)
	}

	return apf.AuthResponse{Authenticated: isValid}
}

// validateCredentials checks the username/password against the device database.
func (h *APFHandler) validateCredentials(username, password string) bool {
	if h.deviceID == "" {
		h.log.Warn("Cannot validate credentials: device ID not set")

		return false
	}

	ctx := context.Background()

	// Fetch device from database using the UUID
	device, err := h.devices.GetByID(ctx, h.deviceID, "", true)
	if err != nil {
		h.log.Warn("Failed to fetch device %s from database: %v", h.deviceID, err)

		return false
	}

	if device == nil {
		h.log.Warn("Device %s not found in database", h.deviceID)

		return false
	}

	// Compare credentials
	// MPSUsername is the field used for CIRA authentication
	if device.MPSUsername != username {
		h.log.Debug("Username mismatch for device %s", h.deviceID)

		return false
	}

	// Compare password
	if device.MPSPassword != password {
		h.log.Debug("Password mismatch for device %s", h.deviceID)

		return false
	}

	return true
}

// OnGlobalRequest is called when an APF_GLOBAL_REQUEST message is received.
// Tracks TCP forwarding requests and returns true when keep-alive should be sent.
func (h *APFHandler) OnGlobalRequest(request apf.GlobalRequest) bool {
	h.globalRequestCount++

	h.log.Debug("Global request %d - Type: %s, Address: %s, Port: %d",
		h.globalRequestCount, request.RequestType, request.Address, request.Port)

	// Send keep-alive options after the threshold is reached
	// This is when the CIRA connection setup is complete
	return h.globalRequestCount >= globalRequestThreshold
}

// ShouldSendKeepAlive returns whether keep-alive should be sent based on global request count.
func (h *APFHandler) ShouldSendKeepAlive() bool {
	return h.globalRequestCount >= globalRequestThreshold
}
