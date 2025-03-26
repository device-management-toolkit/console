package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/open-amt-cloud-toolkit/console/internal/entity/dto/v1"
)

func (r *deviceManagementRoutes) getVersion(c *gin.Context) {
	guid := c.Param("guid")

	versionv1, _, err := r.d.GetVersion(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - GetVersion")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, versionv1)
}

func (r *deviceManagementRoutes) getFeatures(c *gin.Context) {
	guid := c.Param("guid")

	features, _, err := r.d.GetFeatures(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - getFeatures")
		ErrorResponse(c, err)

		return
	}

	v1Features := map[string]interface{}{
		"redirection":           features.Redirection,
		"KVM":                   features.EnableKVM,
		"SOL":                   features.EnableSOL,
		"IDER":                  features.EnableIDER,
		"optInState":            features.OptInState,
		"userConsent":           features.UserConsent,
		"kvmAvailable":          features.KVMAvailable,
		"ocr":                   features.OCR,
		"httpsBootSupported":    features.HTTPSBootSupported,
		"winREBootSupported":    features.WinREBootSupported,
		"localPBABootSupported": features.LocalPBABootSupported,
		"remoteErase":           features.RemoteErase,
	}

	c.JSON(http.StatusOK, v1Features)
}

func (r *deviceManagementRoutes) setFeatures(c *gin.Context) {
	guid := c.Param("guid")

	var features dto.Features
	if err := c.ShouldBindJSON(&features); err != nil {
		ErrorResponse(c, err)

		return
	}

	features, _, err := r.d.SetFeatures(c.Request.Context(), guid, features)
	if err != nil {
		r.l.Error(err, "http - v1 - setFeatures")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, features)
}
