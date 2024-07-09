package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/open-amt-cloud-toolkit/console/internal/entity/dto"
	"github.com/open-amt-cloud-toolkit/console/internal/usecase/ieee8021xconfigs"
	"github.com/open-amt-cloud-toolkit/console/pkg/consoleerrors"
	"github.com/open-amt-cloud-toolkit/console/pkg/logger"
)

var ErrValidation8021xConfig = dto.NotValidError{Console: consoleerrors.CreateConsoleError("8021xConfigAPI")}

type ieee8021xConfigRoutes struct {
	t ieee8021xconfigs.Feature
	l logger.Interface
}

func NewIEEE8021xConfigRoutes(handler *gin.RouterGroup, t ieee8021xconfigs.Feature, l logger.Interface) {
	r := &ieee8021xConfigRoutes{t, l}

	if binding.Validator != nil {
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			err := v.RegisterValidation("authProtocolValidator", dto.AuthProtocolValidator)
			if err != nil {
				l.Error(err, "failed to register validation")
			}
		}
	}

	h := handler.Group("/ieee8021xconfigs")
	{
		h.GET("", r.get)
		h.GET(":profileName", r.getByName)
		h.POST("", r.insert)
		h.PATCH("", r.update)
		h.DELETE(":profileName", r.delete)
	}
}

type IEEE8021xConfigCountResponse struct {
	Count int                   `json:"totalCount"`
	Data  []dto.IEEE8021xConfig `json:"data"`
}

// @Summary     Get All IEEE802.1x Configs
// @Description Retrieves all of the IEEE802.1x configuration profiles from the database. 
// @ID          get8021xConfigs
// @Tags  	    ieee802.1x
// @Accept      json
// @Produce     json
// @Success     200 {object} []dto.IEEE8021xConfig
// @Failure     500 {object} response
// @Router      /api/v1/admin/ieee8021xconfigs/ [get]
func (r *ieee8021xConfigRoutes) get(c *gin.Context) {
	var odata OData
	if err := c.ShouldBindQuery(&odata); err != nil {
		validationErr := ErrValidation8021xConfig.Wrap("get", "ShouldBindQuery", err)
		ErrorResponse(c, validationErr)

		return
	}

	items, err := r.t.Get(c.Request.Context(), odata.Top, odata.Skip, "")
	if err != nil {
		r.l.Error(err, "http - IEEE8021x configs - v1 - getCount")
		ErrorResponse(c, err)

		return
	}

	if odata.Count {
		count, err := r.t.GetCount(c.Request.Context(), "")
		if err != nil {
			r.l.Error(err, "http - IEEE8021x configs - v1 - getCount")
			ErrorResponse(c, err)
		}

		countResponse := IEEE8021xConfigCountResponse{
			Count: count,
			Data:  items,
		}

		c.JSON(http.StatusOK, countResponse)
	} else {
		c.JSON(http.StatusOK, items)
	}
}

// @Summary     Get a IEEE802.1x Config
// @Description Retrieves the specific IEEE802.1x configuration profile from the database. 
// @ID          get8021xConfig
// @Tags  	    ieee802.1x
// @Accept      json
// @Produce     json
// @Success     200 {object} dto.IEEE8021xConfig
// @Failure     500 {object} response
// @Router      /api/v1/admin/ieee8021xconfigs/:profileName [get]
func (r *ieee8021xConfigRoutes) getByName(c *gin.Context) {
	configName := c.Param("profileName")

	config, err := r.t.GetByName(c.Request.Context(), configName, "")
	if err != nil {
		r.l.Error(err, "http - IEEE8021x configs - v1 - getByName")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, config)
}

// @Summary     Create an IEEE802.1x Config
// @Description Creates a new IEEE802.1x configuration profile. 
// @ID          add8021xConfig
// @Tags  	    ieee802.1x
// @Accept      json
// @Produce     json
// @Success     200 {object} dto.IEEE8021xConfig
// @Failure     500 {object} response
// @Router      /api/v1/admin/ieee8021xconfigs/ [post]
func (r *ieee8021xConfigRoutes) insert(c *gin.Context) {
	var config dto.IEEE8021xConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		validationErr := ErrValidation8021xConfig.Wrap("insert", "ShouldBindJSON", err)
		ErrorResponse(c, validationErr)

		return
	}

	newConfig, err := r.t.Insert(c.Request.Context(), &config)
	if err != nil {
		r.l.Error(err, "http - IEEE8021x configs - v1 - insert")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusCreated, newConfig)
}

// @Summary     Edit a IEEE802.1x Config
// @Description Edits an existing IEEE802.1x configuration profile.
// @Description 
// @Description The profileName can not be changed.
// @Description 
// @Description Version must be provided to ensure the correct profile is edited.
// @ID          edit8021xConfig
// @Tags  	    ieee802.1x
// @Accept      json
// @Produce     json
// @Success     200 {object} dto.IEEE8021xConfig
// @Failure     500 {object} response
// @Router      /api/v1/admin/ieee8021xconfigs/ [patch]
func (r *ieee8021xConfigRoutes) update(c *gin.Context) {
	var config dto.IEEE8021xConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		validationErr := ErrValidation8021xConfig.Wrap("update", "ShouldBindJSON", err)
		ErrorResponse(c, validationErr)

		return
	}

	updatedConfig, err := r.t.Update(c.Request.Context(), &config)
	if err != nil {
		r.l.Error(err, "http - IEEE8021x configs - v1 - update")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, updatedConfig)
}

// @Summary     Remove a IEEE802.1x Config
// @Description Removes the specific IEEE802.1x configuration profile from the database. 
// @ID          delete8021xConfig
// @Tags  	    ieee802.1x
// @Accept      json
// @Produce     json
// @Success     200 {object} nil
// @Failure     500 {object} response
// @Router      /api/v1/admin/ieee8021xconfigs/:profileName [delete]
func (r *ieee8021xConfigRoutes) delete(c *gin.Context) {
	configName := c.Param("profileName")

	err := r.t.Delete(c.Request.Context(), configName, "")
	if err != nil {
		r.l.Error(err, "http - IEEE8021x configs - v1 - delete")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusNoContent, nil)
}
