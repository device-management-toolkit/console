package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/packaging"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const unknownContentLength int64 = -1

var errValidationPackage = dto.NotValidError{Console: consoleerrors.CreateConsoleError("PackageAPI")}

type packageRoutes struct {
	t packaging.Feature
	l logger.Interface
}

// NewPackageRoutes registers the Download RPC endpoints under the given group.
func NewPackageRoutes(handler *gin.RouterGroup, t packaging.Feature, l logger.Interface) {
	r := &packageRoutes{t: t, l: l}

	h := handler.Group("/package")
	{
		h.GET("/rpc-versions", r.versions)
		h.POST("", r.build)
	}
}

func (r *packageRoutes) versions(c *gin.Context) {
	releases, err := r.t.ListVersions(c.Request.Context())
	if err != nil {
		r.l.Error(err, "http - v1 - package - rpc-versions")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, releases)
}

func (r *packageRoutes) build(c *gin.Context) {
	var req dto.PackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErr := errValidationPackage.Wrap("build", "ShouldBindJSON", err)
		ErrorResponse(c, validationErr)

		return
	}

	reader, filename, err := r.t.BuildPackage(c.Request.Context(), req)
	if err != nil {
		r.l.Error(err, "http - v1 - package - build")
		ErrorResponse(c, err)

		return
	}

	c.DataFromReader(http.StatusOK, unknownContentLength, "application/zip", reader, map[string]string{
		"Content-Disposition": `attachment; filename="` + filename + `"`,
	})
}
