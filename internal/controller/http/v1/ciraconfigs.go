package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/ciraconfigs"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type ciraConfigRoutes struct {
	cira ciraconfigs.Feature
	l    logger.Interface
}

func NewCIRAConfigRoutes(handler *gin.RouterGroup, t ciraconfigs.Feature, l logger.Interface) {
	r := &ciraConfigRoutes{t, l}

	h := handler.Group("/ciraconfigs")
	{
		h.GET("", r.get)
		h.GET(":ciraConfigName", r.getByName)
		h.POST("", r.insert)
		h.PATCH("", r.update)
		h.DELETE(":ciraConfigName", r.delete)
	}
}

func (r *ciraConfigRoutes) get(c *gin.Context) {
	var odata OData
	if err := c.ShouldBindQuery(&odata); err != nil {
		r.l.Error(err, "http - CIRA configs - v1 - getCount")
		ErrorResponse(c, err)

		return
	}

	configs, err := r.cira.Get(c.Request.Context(), odata.Top, odata.Skip, "")
	if err != nil {
		r.l.Error(err, "http - CIRA configs - v1 - getCount")
		ErrorResponse(c, err)

		return
	}

	if odata.Count {
		count, err := r.cira.GetCount(c.Request.Context(), "")
		if err != nil {
			r.l.Error(err, "http - CIRA configs - v1 - getCount")
			ErrorResponse(c, err)

			return
		}

		countResponse := dto.CIRAConfigCountResponse{
			Count: count,
			Data:  configs,
		}

		c.JSON(http.StatusOK, countResponse)
	} else {
		c.JSON(http.StatusOK, configs)
	}
}

func (r *ciraConfigRoutes) getByName(c *gin.Context) {
	configName := c.Param("ciraConfigName")

	foundConfig, err := r.cira.GetByName(c.Request.Context(), configName, "")
	if err != nil {
		r.l.Error(err, "http - CIRA configs - v1 - getByName")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, foundConfig)
}

func (r *ciraConfigRoutes) insert(c *gin.Context) {
	var config dto.CIRAConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		r.l.Error(err, "http - CIRA configs - v1 - insert")
		ErrorResponse(c, err)

		return
	}

	newCiraConfig, err := r.cira.Insert(c.Request.Context(), &config)
	if err != nil {
		r.l.Error(err, "http - CIRA configs - v1 - insert")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusCreated, newCiraConfig)
}

func (r *ciraConfigRoutes) update(c *gin.Context) {
	var config dto.CIRAConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		r.l.Error(err, "http - CIRA configs - v1 - update")
		ErrorResponse(c, err)

		return
	}

	updatedConfig, err := r.cira.Update(c.Request.Context(), &config)
	if err != nil {
		r.l.Error(err, "http - CIRA configs - v1 - update")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, updatedConfig)
}

func (r *ciraConfigRoutes) delete(c *gin.Context) {
	configName := c.Param("ciraConfigName")

	err := r.cira.Delete(c.Request.Context(), configName, "")
	if err != nil {
		r.l.Error(err, "http - CIRA configs - v1 - delete")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusNoContent, nil)
}
