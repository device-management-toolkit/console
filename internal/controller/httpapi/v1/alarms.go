package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (r *deviceManagementRoutes) getAlarmOccurrences(c *gin.Context) {
	guid := c.Param("guid")
	tenantID := tenantIDFromRequest(c)

	alarms, err := r.d.GetAlarmOccurrences(c.Request.Context(), guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - getFeatures")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, alarms)
}

func (r *deviceManagementRoutes) createAlarmOccurrences(c *gin.Context) {
	guid := c.Param("guid")

	alarm := &dto.AlarmClockOccurrenceInput{}
	if err := c.ShouldBindJSON(alarm); err != nil {
		ErrorResponse(c, err)

		return
	}

	tenantID := tenantIDFromRequest(c)

	alarmReference, err := r.d.CreateAlarmOccurrences(c.Request.Context(), guid, tenantID, *alarm)
	if err != nil {
		r.l.Error(err, "http - v1 - createAlarmOccurrences")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusCreated, alarmReference)
}

func (r *deviceManagementRoutes) deleteAlarmOccurrences(c *gin.Context) {
	guid := c.Param("guid")

	alarm := dto.DeleteAlarmOccurrenceRequest{}
	if err := c.ShouldBindJSON(&alarm); err != nil {
		ErrorResponse(c, err)

		return
	}

	tenantID := tenantIDFromRequest(c)

	err := r.d.DeleteAlarmOccurrences(c.Request.Context(), guid, alarm.Name, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - deleteAlarmOccurrences")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusNoContent, nil)
}
