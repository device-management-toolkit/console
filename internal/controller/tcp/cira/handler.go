package cira

import (
	"context"
	"log"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/apf"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
)

// APFHandler implements apf.Handler for the CIRA server.
// It provides application-specific logic for authentication and device registration.
type APFHandler struct {
	devices            devices.Feature
	deviceID           string
	globalRequestCount int
}

// NewAPFHandler creates a new APF handler with access to the devices feature.
func NewAPFHandler(d devices.Feature) *APFHandler {
	return &APFHandler{
		devices: d,
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

	log.Printf("APF Protocol Version - Version: %d.%d, Trigger: %d, UUID: %s\n",
		info.MajorVersion, info.MinorVersion, info.TriggerReason, info.UUID)

	return nil
}

// OnAuthRequest is called when an APF_USERAUTH_REQUEST message is received.
// Validates credentials against the database.
func (h *APFHandler) OnAuthRequest(request apf.AuthRequest) apf.AuthResponse {
	log.Printf("Authentication attempt - Device: %s, Username: %s, Method: %s\n",
		h.deviceID, request.Username, request.MethodName)

	// Only support password authentication
	if request.MethodName != "password" {
		log.Printf("Unsupported authentication method: %s\n", request.MethodName)

		return apf.AuthResponse{Authenticated: false}
	}

	// Validate credentials against database
	isValid := h.validateCredentials(request.Username, request.Password)

	if isValid {
		log.Printf("Authentication successful for device %s\n", h.deviceID)
	} else {
		log.Printf("Authentication failed for device %s with username %s\n",
			h.deviceID, request.Username)
	}

	return apf.AuthResponse{Authenticated: isValid}
}

// validateCredentials checks the username/password against the device database.
func (h *APFHandler) validateCredentials(username, password string) bool {
	if h.deviceID == "" {
		log.Println("Cannot validate credentials: device ID not set")

		return false
	}

	ctx := context.Background()

	// Fetch device from database using the UUID
	device, err := h.devices.GetByID(ctx, h.deviceID, "", true)
	if err != nil {
		log.Printf("Failed to fetch device %s from database: %v\n", h.deviceID, err)

		return false
	}

	if device == nil {
		log.Printf("Device %s not found in database\n", h.deviceID)

		return false
	}

	// Compare credentials
	// MPSUsername is the field used for CIRA authentication
	if device.MPSUsername != username {
		log.Printf("Username mismatch for device %s\n", h.deviceID)

		return false
	}

	// Compare password
	if device.Password != password {
		log.Printf("Password mismatch for device %s\n", h.deviceID)
		// TODO: set back to false
		return true
	}

	return true
}

// OnGlobalRequest is called when an APF_GLOBAL_REQUEST message is received.
// Tracks TCP forwarding requests and returns true when keep-alive should be sent.
func (h *APFHandler) OnGlobalRequest(request apf.GlobalRequest) bool {
	h.globalRequestCount++

	log.Printf("Global request %d - Type: %s, Address: %s, Port: %d\n",
		h.globalRequestCount, request.RequestType, request.Address, request.Port)

	// Send keep-alive options after the fourth global request
	// This is when the CIRA connection setup is complete
	// TODO: Make the threshold configurable from console config
	return h.globalRequestCount >= 4
}

// ShouldSendKeepAlive returns whether keep-alive should be sent based on global request count.
func (h *APFHandler) ShouldSendKeepAlive() bool {
	return h.globalRequestCount >= 4
}
