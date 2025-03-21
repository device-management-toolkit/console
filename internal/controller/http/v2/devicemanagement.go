package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/open-amt-cloud-toolkit/console/internal/controller/http/v1"
	"github.com/open-amt-cloud-toolkit/console/internal/entity/dto/v1"
	"github.com/open-amt-cloud-toolkit/console/internal/usecase/devices"
	"github.com/open-amt-cloud-toolkit/console/pkg/logger"
)

type deviceManagementRoutes struct {
	d devices.Feature
	l logger.Interface
}

func NewAmtRoutes(handler *gin.RouterGroup, d devices.Feature, l logger.Interface) {
	r := &deviceManagementRoutes{d, l}

	h := handler.Group("/amt")
	{
		h.GET("version/:guid", r.getVersion)
		h.GET("features/:guid", r.getFeatures)
		h.POST("features/:guid", r.setFeatures)
		h.GET("hardwareInfo/:guid", r.getHardwareInfo)
	}
}

func (r *deviceManagementRoutes) getVersion(c *gin.Context) {
	guid := c.Param("guid")

	_, v2, err := r.d.GetVersion(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v2 - GetVersion")
		v1.ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, v2)
}

func (r *deviceManagementRoutes) getFeatures(c *gin.Context) {
	guid := c.Param("guid")

	_, v2, err := r.d.GetFeatures(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v2 - getFeatures")
		v1.ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, v2)
}

func (r *deviceManagementRoutes) setFeatures(c *gin.Context) {
	guid := c.Param("guid")

	var features dto.Features

	if err := c.ShouldBindJSON(&features); err != nil {
		r.l.Error(err, "http - v2 - setFeatures")
		v1.ErrorResponse(c, err)

		return
	}

	_, v2, err := r.d.SetFeatures(c.Request.Context(), guid, features)
	if err != nil {
		r.l.Error(err, "http - v2 - setFeatures")
		v1.ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, v2)
}

func (r *deviceManagementRoutes) getHardwareInfo(c *gin.Context) {
	guid := c.Param("guid")

	_, hwInfo, err := r.d.GetHardwareInfo(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v2 - getHardwareInfo")
		v1.ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, hwInfo)
}
