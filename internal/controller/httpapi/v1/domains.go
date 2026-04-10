package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var ErrValidationDomains = dto.NotValidError{Console: consoleerrors.CreateConsoleError("DomainsAPI")}

type domainRoutes struct {
	t domains.Feature
	l logger.Interface
}

func NewDomainRoutes(handler *gin.RouterGroup, t domains.Feature, l logger.Interface) {
	r := &domainRoutes{t, l}

	h := handler.Group("/domains")
	{
		h.GET("", r.get)
		h.GET(":name", r.getByName)
		h.POST("", r.insert)
		h.PATCH("", r.update)
		h.DELETE(":name", r.delete)
	}
}

type DomainCountResponse struct {
	Count int          `json:"totalCount"`
	Data  []dto.Domain `json:"data"`
}

func (r *domainRoutes) get(c *gin.Context) {
	var odata OData
	if err := c.ShouldBindQuery(&odata); err != nil {
		validationErr := ErrValidationDomains.Wrap("get", "ShouldBindQuery", err)
		ErrorResponse(c, validationErr)

		return
	}

	items, err := r.t.Get(c.Request.Context(), odata.Top, odata.Skip, "")
	if err != nil {
		r.l.Error(err, "http - v1 - getCount")
		ErrorResponse(c, err)

		return
	}

	if odata.Count {
		count, err := r.t.GetCount(c.Request.Context(), "")
		if err != nil {
			r.l.Error(err, "http - v1 - getCount")
			ErrorResponse(c, err)
		}

		countResponse := DomainCountResponse{
			Count: count,
			Data:  items,
		}

		c.JSON(http.StatusOK, countResponse)
	} else {
		c.JSON(http.StatusOK, items)
	}
}

func (r *domainRoutes) getByName(c *gin.Context) {
	name := c.Param("name")

	item, err := r.t.GetByName(c.Request.Context(), name, "")
	if err != nil {
		r.l.Error(err, "http - v1 - getByName")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, item)
}

func (r *domainRoutes) insert(c *gin.Context) {
	var domain dto.Domain
	if err := c.ShouldBindJSON(&domain); err != nil {
		validationErr := ErrValidationDomains.Wrap("insert", "ShouldBindJSON", err)
		ErrorResponse(c, validationErr)

		return
	}

	newDomain, err := r.t.Insert(c.Request.Context(), &domain)
	if err != nil {
		r.l.Error(err, "http - v1 - insert")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusCreated, newDomain)
}

func (r *domainRoutes) update(c *gin.Context) {
	var domain dto.Domain
	if err := c.ShouldBindJSON(&domain); err != nil {
		validationErr := ErrValidationDomains.Wrap("update", "ShouldBindJSON", err)
		ErrorResponse(c, validationErr)

		return
	}

	updatedDomain, err := r.t.Update(c.Request.Context(), &domain)
	if err != nil {
		r.l.Error(err, "http - v1 - update")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, updatedDomain)
}

func (r *domainRoutes) delete(c *gin.Context) {
	name := c.Param("name")

	err := r.t.Delete(c.Request.Context(), name, "")
	if err != nil {
		r.l.Error(err, "http - v1 - delete")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusNoContent, nil)
}
