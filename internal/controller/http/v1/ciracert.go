package v1

import (
	"encoding/pem"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/pkg/logger"
)

type ciraCertRoutes struct {
	l logger.Interface
}

func NewCIRACertRoutes(handler *gin.RouterGroup, l logger.Interface) {
	r := &ciraCertRoutes{l}

	h := handler.Group("/ciracert")
	{
		h.GET("", r.getCIRACert)
	}
}

func (r *ciraCertRoutes) getCIRACert(c *gin.Context) {
	// Read the root certificate file
	certData, err := os.ReadFile("config/root_cert.pem")
	if err != nil {
		r.l.Error(err, "http - CIRA cert - v1 - getCIRACert - failed to read certificate file")
		c.String(http.StatusInternalServerError, "Failed to read certificate file")
		return
	}

	// Decode the PEM block
	block, _ := pem.Decode(certData)
	if block == nil {
		r.l.Error(nil, "http - CIRA cert - v1 - getCIRACert - failed to decode PEM")
		c.String(http.StatusInternalServerError, "Failed to decode certificate")
		return
	}

	// Extract just the base64-encoded certificate data (strip PEM headers/footers)
	pemString := string(certData)
	lines := strings.Split(pemString, "\n")
	var certContent strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and PEM headers/footers
		if trimmed == "" || strings.HasPrefix(trimmed, "-----") {
			continue
		}
		certContent.WriteString(trimmed)
	}

	// Return as plain text
	c.String(http.StatusOK, certContent.String())
}
