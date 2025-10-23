package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/profiles"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var ErrValidationProfile = dto.NotValidError{Console: consoleerrors.CreateConsoleError("ProfileAPI")}

type profileRoutes struct {
	t profiles.Feature
	l logger.Interface
}

func NewProfileRoutes(handler *gin.RouterGroup, t profiles.Feature, l logger.Interface) {
	r := &profileRoutes{t, l}

	if binding.Validator != nil {
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			_ = v.RegisterValidation("genpasswordwone", dto.ValidateAMTPassOrGenRan)
			_ = v.RegisterValidation("ciraortls", dto.ValidateCIRAOrTLS)
		}
	}

	h := handler.Group("/profiles")
	{
		h.GET("", r.get)
		h.GET(":name", r.getByName)
		h.POST("", r.insert)
		h.PATCH("", r.update)
		h.DELETE(":name", r.delete)
		h.GET("export/:name", r.export)
	}
}

// @Summary     Show Profiles
// @Description Show all profiles
// @ID          profiles
// @Tags  	    profiles
// @Accept      json
// @Produce     json
// @Success     200 {object} ProfileCountResponse
// @Failure     500 {object} response
// @Router      /api/v1/admin/profiles [get]
func (r *profileRoutes) get(c *gin.Context) {
	var odata OData
	if err := c.ShouldBindQuery(&odata); err != nil {
		validationErr := ErrValidationProfile.Wrap("get", "ShouldBindQuery", err)
		ErrorResponse(c, validationErr)

		return
	}

	items, err := r.t.Get(c.Request.Context(), odata.Top, odata.Skip, "")
	if err != nil {
		r.l.Error(err, "http - v1 - get")
		ErrorResponse(c, err)

		return
	}

	if odata.Count {
		count, err := r.t.GetCount(c.Request.Context(), "")
		if err != nil {
			r.l.Error(err, "http - v1 - getCount")
			ErrorResponse(c, err)
		}

		countResponse := dto.ProfileCountResponse{
			Count: count,
			Data:  items,
		}

		c.JSON(http.StatusOK, countResponse)
	} else {
		c.JSON(http.StatusOK, items)
	}
}

// @Summary     Show Profile
// @Description Show profile by name
// @ID          profile
// @Tags              profiles
// @Accept      json
// @Produce     json
// @Success     200 {object} dto.Profile
// @Failure     500 {object} response
// @Router      /api/v1/admin/profiles/:name [get]

func (r *profileRoutes) getByName(c *gin.Context) {
	name := c.Param("name")

	item, err := r.t.GetByName(c.Request.Context(), name, "")
	if err != nil {
		r.l.Error(err, "http - v1 - getByName")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, item)
}

// @Summary     Export Profile
// @Description Export profile by name
// @ID          export-profile
// @Tags              profiles
// @Accept      json
// @Produce     json
// @Success     200 {object} dto.ProfileExportResponse
// @Failure     500 {object} response
// @Router      /api/v1/admin/profiles/export/:name [get]

func (r *profileRoutes) export(c *gin.Context) {
	name := c.Param("name")
	domainName := c.Query("domainName")

	item, key, err := r.t.Export(c.Request.Context(), name, domainName, "")
	if err != nil {
		r.l.Error(err, "http - v1 - export")
		ErrorResponse(c, err)

		return
	}

	// Create response using the DTO type
	response := dto.ProfileExportResponse{
		Filename: name + ".yaml",
		Content:  item,
		Key:      key,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary     Add Profile
// @Description Add Profile
// @ID          profiles
// @Tags              profiles
// @Accept      json
// @Produce     json
// @Success     200 {object} ProfileResponse
// @Failure     500 {object} response
// @Router      /api/v1/admin/profiles [post]

func (r *profileRoutes) insert(c *gin.Context) {
	var profile dto.Profile
	if err := c.ShouldBindJSON(&profile); err != nil {
		validationErr := ErrValidationProfile.Wrap("insert", "ShouldBindJSON", err)
		ErrorResponse(c, validationErr)

		return
	}

	newProfile, err := r.t.Insert(c.Request.Context(), &profile)
	if err != nil {
		r.l.Error(err, "http - v1 - insert")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusCreated, newProfile)
}

// @Summary     Edit Profile
// @Description Edit a Profile
// @ID          updateProfile
// @Tags              profiles
// @Accept      json
// @Produce     json
// @Success     200 {object} ProfileResponse
// @Failure     500 {object} response
// @Router      /api/v1/admin/Profiles [patch]

func (r *profileRoutes) update(c *gin.Context) {
	var profile dto.Profile
	if err := c.ShouldBindJSON(&profile); err != nil {
		validationErr := ErrValidationProfile.Wrap("update", "ShouldBindJSON", err)
		ErrorResponse(c, validationErr)

		return
	}

	updatedProfile, err := r.t.Update(c.Request.Context(), &profile)
	if err != nil {
		r.l.Error(err, "http - v1 - update")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, updatedProfile)
}

// @Summary     Remove Profiles
// @Description Remove a Profile
// @ID          deleteProfile
// @Tags              profiles
// @Accept      json
// @Produce     json
// @Success     204 {object} noContent
// @Failure     500 {object} response
// @Router      /api/v1/admin/profiles [delete]

func (r *profileRoutes) delete(c *gin.Context) {
	name := c.Param("name")

	err := r.t.Delete(c.Request.Context(), name, "")
	if err != nil {
		r.l.Error(err, "http - v1 - delete")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusNoContent, nil)
}
