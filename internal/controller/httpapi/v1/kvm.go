package v1

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
)

// ErrKVMCaptureUnsupported is returned when the wired usecase does not implement KVM frame capture.
var ErrKVMCaptureUnsupported = errors.New("kvm frame capture is not supported by this build")

// kvmFrameCapturer is the subset of the devices usecase needed to capture a KVM frame.
// It is resolved via type assertion so the shared Feature interface (and its mocks) stay unchanged.
type kvmFrameCapturer interface {
	CaptureKVMFrame(ctx context.Context, guid string, opts devices.CaptureKVMOptions) (dto.KVMFrame, error)
}

// getKVMDisplays returns current IPS_ScreenSettingData for the device
func (r *deviceManagementRoutes) getKVMDisplays(c *gin.Context) {
	guid := c.Param("guid")

	settings, err := r.d.GetKVMScreenSettings(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - getKVMDisplays")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, settings)
}

// setKVMDisplays updates IPS_ScreenSettingData for the device
func (r *deviceManagementRoutes) setKVMDisplays(c *gin.Context) {
	guid := c.Param("guid")

	var req dto.KVMScreenSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, err)

		return
	}

	settings, err := r.d.SetKVMScreenSettings(c.Request.Context(), guid, req)
	if err != nil {
		r.l.Error(err, "http - v1 - setKVMDisplays")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, settings)
}

// captureKVMFrame captures a single raw KVM screen frame for external (agent/VLM) decoding.
func (r *deviceManagementRoutes) captureKVMFrame(c *gin.Context) {
	guid := c.Param("guid")

	capturer, ok := r.d.(kvmFrameCapturer)
	if !ok {
		ErrorResponse(c, ErrKVMCaptureUnsupported)

		return
	}

	opts := devices.CaptureKVMOptions{
		MaxWidth:       queryInt(c, "width"),
		MaxHeight:      queryInt(c, "height"),
		TimeoutSeconds: queryInt(c, "timeoutSeconds"),
	}

	frame, err := capturer.CaptureKVMFrame(c.Request.Context(), guid, opts)
	if err != nil {
		r.l.Error(err, "http - v1 - captureKVMFrame")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, frame)
}

// queryInt reads an optional non-negative integer query parameter, defaulting to 0.
func queryInt(c *gin.Context, key string) int {
	v, err := strconv.Atoi(c.Query(key))
	if err != nil || v < 0 {
		return 0
	}

	return v
}
