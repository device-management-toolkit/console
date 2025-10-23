package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type deviceRoutes struct {
	t devices.Feature
	l logger.Interface
}

var ErrValidationDevices = dto.NotValidError{Console: consoleerrors.CreateConsoleError("ProfileAPI")}

func NewDeviceRoutes(handler *gin.RouterGroup, t devices.Feature, l logger.Interface) {
	r := &deviceRoutes{t, l}

	handler.GET("authorize/redirection/:id", r.LoginRedirection)

	h := handler.Group("/devices")
	{
		h.GET("", r.get)
		h.GET("stats", r.getStats)
		h.GET("redirectstatus/:guid", r.redirectStatus)
		h.GET("cert/:guid", r.getDeviceCertificate)
		h.POST("cert/:guid", r.pinDeviceCertificate)
		h.DELETE("cert/:guid", r.deleteDeviceCertificate)
		h.GET(":guid", r.getByID)
		h.GET("tags", r.getTags)
		h.POST("", r.insert)
		h.PATCH("", r.update)
		h.DELETE(":guid", r.delete)
	}
}

func (dr *deviceRoutes) getStats(c *gin.Context) {
	count, err := dr.t.GetCount(c.Request.Context(), "")
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - getCount")
		ErrorResponse(c, err)

		return
	}

	countResponse := dto.DeviceStatResponse{
		TotalCount: count,
	}

	c.JSON(http.StatusOK, countResponse)
}

func (dr *deviceRoutes) LoginRedirection(c *gin.Context) {
	deviceID := c.Param("id")

	_, err := dr.t.GetByID(c.Request.Context(), deviceID, "", false)
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - LoginRedirection")
		ErrorResponse(c, err)

		return
	}
	// Create JWT token
	expirationTime := time.Now().Add(config.ConsoleConfig.JWTExpiration)
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expirationTime),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(config.ConsoleConfig.JWTKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})

		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

func (dr *deviceRoutes) get(c *gin.Context) {
	var odata OData
	if err := c.ShouldBindQuery(&odata); err != nil {
		ErrorResponse(c, err)

		return
	}

	tags := c.Query("tags")
	hostname := c.Query("hostname")
	friendlyName := c.Query("friendlyName")

	var items []dto.Device

	var err error

	switch {
	case hostname != "":
		items, err = dr.getByColumnOrTags(c, "HostName", hostname, odata.Top, odata.Skip, "")

	case friendlyName != "":
		items, err = dr.getByColumnOrTags(c, "FriendlyName", friendlyName, odata.Top, odata.Skip, "")

	case tags != "":
		items, err = dr.getByColumnOrTags(c, "Tags", tags, odata.Top, odata.Skip, "")

	default:
		items, err = dr.t.Get(c.Request.Context(), odata.Top, odata.Skip, "")
	}

	if err != nil {
		dr.l.Error(err, "http - devices - v1 - get")
		ErrorResponse(c, err)

		return
	}

	if odata.Count {
		count, err := dr.t.GetCount(c.Request.Context(), "")
		if err != nil {
			dr.l.Error(err, "http - devices - v1 - get")
			ErrorResponse(c, err)

			return
		}

		countResponse := dto.DeviceCountResponse{
			Count: count,
			Data:  items,
		}

		c.JSON(http.StatusOK, countResponse)
	} else {
		c.JSON(http.StatusOK, items)
	}
}

func (dr *deviceRoutes) getByColumnOrTags(c *gin.Context, column, value string, limit, skip int, tenantID string) ([]dto.Device, error) {
	var items []dto.Device

	var err error

	ctx := c.Request.Context()
	if column == "Tags" {
		items, err = dr.t.GetByTags(ctx, value, c.Query("method"), limit, skip, tenantID)
	} else {
		items, err = dr.t.GetByColumn(ctx, column, value, "")
	}

	if err != nil {
		return nil, err
	}

	return items, nil
}

func (dr *deviceRoutes) getByID(c *gin.Context) {
	var odata OData
	if err := c.ShouldBindQuery(&odata); err != nil {
		ErrorResponse(c, err)

		return
	}

	guid := c.Param("guid")

	item, err := dr.t.GetByID(c.Request.Context(), guid, "", false)
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - get")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, item)
}

func (dr *deviceRoutes) insert(c *gin.Context) {
	var device dto.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		validationErr := ErrValidationDevices.Wrap("insert", "ShouldBindJSON", err)
		ErrorResponse(c, validationErr)

		return
	}

	newDevice, err := dr.t.Insert(c.Request.Context(), &device)
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - insert")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusCreated, newDevice)
}

func (dr *deviceRoutes) update(c *gin.Context) {
	var device dto.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		ErrorResponse(c, err)

		return
	}

	updatedDevice, err := dr.t.Update(c.Request.Context(), &device)
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - update")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, updatedDevice)
}

func (dr *deviceRoutes) delete(c *gin.Context) {
	guid := c.Param("guid")

	err := dr.t.Delete(c.Request.Context(), guid, "")
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - delete")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (dr *deviceRoutes) redirectStatus(c *gin.Context) {
	_ = c.Param("guid")
	result := map[string]bool{
		"isSOLConnected":  false, // device.solConnect,
		"isIDERConnected": false, // device.iderConnect,
	}
	c.JSON(http.StatusOK, result)
}

func (dr *deviceRoutes) getTags(c *gin.Context) {
	tags, err := dr.t.GetDistinctTags(c.Request.Context(), "")
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - tags")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, tags)
}

func (dr *deviceRoutes) getDeviceCertificate(c *gin.Context) {
	var odata OData
	if err := c.ShouldBindQuery(&odata); err != nil {
		ErrorResponse(c, err)

		return
	}

	guid := c.Param("guid")

	item, err := dr.t.GetByID(c.Request.Context(), guid, "", false)
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - cert")
		ErrorResponse(c, err)

		return
	}

	cert, err := dr.t.GetDeviceCertificate(c.Request.Context(), item.GUID)
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - cert")
		ErrorResponse(c, err)

		return
	}

	cert.GUID = item.GUID

	c.JSON(http.StatusOK, cert)
}

func (dr *deviceRoutes) pinDeviceCertificate(c *gin.Context) {
	var certToPin dto.PinCertificate
	if err := c.ShouldBindBodyWithJSON(&certToPin); err != nil {
		ErrorResponse(c, err)

		return
	}

	guid := c.Param("guid")

	item, err := dr.t.GetByID(c.Request.Context(), guid, "", true)
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - deleteDeviceCertificate - getById")
		ErrorResponse(c, err)

		return
	}

	item.CertHash = certToPin.SHA256Fingerprint

	item, err = dr.t.Update(c.Request.Context(), item)
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - deleteDeviceCertificate - update")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, item)
}

func (dr *deviceRoutes) deleteDeviceCertificate(c *gin.Context) {
	var odata OData
	if err := c.ShouldBindQuery(&odata); err != nil {
		ErrorResponse(c, err)

		return
	}

	guid := c.Param("guid")

	item, err := dr.t.GetByID(c.Request.Context(), guid, "", true)
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - deleteDeviceCertificate - getById")
		ErrorResponse(c, err)

		return
	}

	item.CertHash = ""

	item, err = dr.t.Update(c.Request.Context(), item)
	if err != nil {
		dr.l.Error(err, "http - devices - v1 - deleteDeviceCertificate - update")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, item)
}
