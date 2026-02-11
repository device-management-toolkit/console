// Package v1 provides Redfish v1 API handlers for SessionService
package v1

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	"github.com/device-management-toolkit/console/redfish/internal/entity"
	"github.com/device-management-toolkit/console/redfish/internal/usecase/sessions"
)

const (
	// Base paths
	redfishV1Base        = "/redfish/v1"
	metadataBase         = redfishV1Base + "/$metadata#"
	sessionServicePath   = redfishV1Base + "/SessionService"
	sessionCollectionURL = sessionServicePath + "/Sessions"

	// Session OData metadata constants
	sessionOdataContext      = metadataBase + "Session.Session"
	sessionOdataType         = "#Session.v1_8_0.Session"
	sessionName              = "User Session"
	sessionDescriptionPrefix = sessionName + " for "

	// SessionService OData metadata constants
	sessionServiceOdataContext = metadataBase + "SessionService.SessionService"
	sessionServiceOdataID      = sessionServicePath
	sessionServiceOdataType    = "#SessionService.v1_2_0.SessionService"
	sessionServiceID           = "SessionService"
	sessionServiceName         = "Session Service"
	sessionServiceDescription  = sessionServiceName + " for DMT Console Redfish API"

	// SessionCollection OData metadata constants
	sessionCollectionOdataContext = metadataBase + "SessionCollection.SessionCollection"
	sessionCollectionOdataID      = sessionCollectionURL
	sessionCollectionOdataType    = "#SessionCollection.SessionCollection"
	sessionCollectionName         = "Session Collection"

	// Session path patterns
	sessionServiceSessionsPath = sessionCollectionURL
	sessionBasePath            = sessionCollectionURL + "/"

	// HTTP headers
	headerXAuthToken    = "X-Auth-Token" //nolint:gosec // G101: This is a header name, not a credential
	headerAuthorization = "Authorization"
	headerUserAgent     = "User-Agent"

	// Auth prefixes
	authBearerPrefix = "Bearer "

	// Context keys
	contextKeySession  = "session"
	contextKeyUsername = "username"

	// Status values
	statusEnabled = "Enabled"
	statusOK      = "OK"

	// Error messages
	errMsgInvalidRequestBody       = "Invalid request body"
	errMsgSessionIDRequired        = "Session ID required"
	errMsgUserPassRequired         = "UserName and Password are required"
	errMsgFailedBuildResponse      = "failed to build session response"
	errMsgFailedRetrieveSession    = "failed to retrieve session"
	errMsgFailedDeleteSession      = "failed to delete session"
	errMsgFailedListSessions       = "failed to list sessions"
	errMsgFailedBuildServiceResp   = "failed to build session service response"
	errMsgFailedCreateDescription  = "failed to create description"
	errMsgFailedCreateStatusState  = "failed to create status state"
	errMsgFailedCreateStatusHealth = "failed to create status health"
)

// SessionAuthMiddleware validates X-Auth-Token header
// This middleware integrates DMT JWT tokens with Redfish session authentication
func SessionAuthMiddleware(sessionUseCase *sessions.UseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for X-Auth-Token header (Redfish standard)
		token := c.GetHeader(headerXAuthToken)

		// Fallback to Authorization: Bearer for compatibility
		if token == "" {
			authHeader := c.GetHeader(headerAuthorization)
			if strings.HasPrefix(authHeader, authBearerPrefix) {
				token = strings.TrimPrefix(authHeader, authBearerPrefix)
			}
		}

		if token == "" {
			UnauthorizedError(c)
			c.Abort()

			return
		}

		// Validate token
		session, err := sessionUseCase.ValidateToken(token)
		if err != nil {
			UnauthorizedError(c)
			c.Abort()

			return
		}

		// Store session in context for handlers
		c.Set(contextKeySession, session)
		c.Set(contextKeyUsername, session.Username)

		c.Next()
	}
}

// RedfishServer session endpoint implementations
// These methods are part of RedfishServer to satisfy the generated.ServerInterface

// sessionToRedfishResponse converts entity.Session to generated.SessionSession format.
func sessionToRedfishResponse(s *entity.Session) (*generated.SessionSession, error) {
	context := sessionOdataContext
	odataID := sessionBasePath + s.ID
	odataType := sessionOdataType
	name := sessionName
	descriptionText := sessionDescriptionPrefix + s.Username

	// Create description using union type
	var description generated.SessionSession_Description
	if err := description.FromResourceDescription(descriptionText); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedCreateDescription, err)
	}

	// Create SessionType as Redfish
	var sessionType generated.SessionSession_SessionType
	if err := sessionType.FromSessionSessionTypes(generated.Redfish); err != nil {
		return nil, fmt.Errorf("failed to create session type: %w", err)
	}

	return &generated.SessionSession{
		OdataContext:          &context,
		OdataId:               &odataID,
		OdataType:             &odataType,
		Id:                    s.ID,
		Name:                  name,
		Description:           &description,
		UserName:              &s.Username,
		SessionType:           &sessionType,
		CreatedTime:           &s.CreatedTime,
		ClientOriginIPAddress: &s.ClientIP,
		Password:              nil, // Always null in responses per Redfish spec
		Token:                 nil, // Always null in responses per Redfish spec
	}, nil
}

// buildSessionServiceResponse builds the SessionService response object using generated types
// This method is shared by GET, PATCH, and PUT to avoid duplication
func (r *RedfishServer) buildSessionServiceResponse() (*generated.SessionServiceSessionService, error) {
	context := sessionServiceOdataContext
	odataID := sessionServiceOdataID
	odataType := sessionServiceOdataType
	sessionTimeout := int64(sessions.DefaultSessionTimeout)
	serviceEnabled := true

	// Create description
	var description generated.SessionServiceSessionService_Description
	if err := description.FromResourceDescription(sessionServiceDescription); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedCreateDescription, err)
	}

	// Create status with union types
	var statusState generated.ResourceStatus_State
	if err := statusState.FromResourceStatusState1(statusEnabled); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedCreateStatusState, err)
	}

	var statusHealth generated.ResourceStatus_Health
	if err := statusHealth.FromResourceStatusHealth1(statusOK); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsgFailedCreateStatusHealth, err)
	}

	return &generated.SessionServiceSessionService{
		OdataContext: &context,
		OdataId:      &odataID,
		OdataType:    &odataType,
		Id:           sessionServiceID,
		Name:         sessionServiceName,
		Description:  &description,
		Status: &generated.ResourceStatus{
			State:  &statusState,
			Health: &statusHealth,
		},
		ServiceEnabled: &serviceEnabled,
		SessionTimeout: &sessionTimeout,
		Sessions: &generated.OdataV4IdRef{
			OdataId: StringPtr(sessionServiceSessionsPath),
		},
	}, nil
}

// GetRedfishV1SessionService handles GET /redfish/v1/SessionService
func (r *RedfishServer) GetRedfishV1SessionService(c *gin.Context) {
	SetRedfishHeaders(c)

	response, err := r.buildSessionServiceResponse()
	if err != nil {
		InternalServerError(c, fmt.Errorf("%s: %w", errMsgFailedBuildServiceResp, err))

		return
	}

	c.JSON(http.StatusOK, response)
}

// GetRedfishV1SessionServiceSessions handles GET /redfish/v1/SessionService/Sessions
func (r *RedfishServer) GetRedfishV1SessionServiceSessions(c *gin.Context) {
	SetRedfishHeaders(c)

	sessionList, err := r.SessionUC.ListSessions()
	if err != nil {
		InternalServerError(c, fmt.Errorf("%s: %w", errMsgFailedListSessions, err))

		return
	}

	members := make([]generated.OdataV4IdRef, 0, len(sessionList))
	for _, session := range sessionList {
		members = append(members, generated.OdataV4IdRef{
			OdataId: StringPtr(sessionBasePath + session.ID),
		})
	}

	context := sessionCollectionOdataContext
	odataID := sessionCollectionOdataID
	odataType := sessionCollectionOdataType
	membersCount := int64(len(members))

	response := generated.SessionCollectionSessionCollection{
		OdataContext:      &context,
		OdataId:           &odataID,
		OdataType:         &odataType,
		Name:              sessionCollectionName,
		Members:           &members,
		MembersOdataCount: &membersCount,
	}

	c.JSON(http.StatusOK, response)
}

// PostRedfishV1SessionServiceSessions handles POST /redfish/v1/SessionService/Sessions
func (r *RedfishServer) PostRedfishV1SessionServiceSessions(c *gin.Context) {
	SetRedfishHeaders(c)

	// Parse request body
	var request struct {
		UserName string `json:"UserName" binding:"required"`
		Password string `json:"Password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequestError(c, errMsgInvalidRequestBody)

		return
	}

	// Get client info
	clientIP := c.ClientIP()
	userAgent := c.GetHeader(headerUserAgent)

	// Create session
	session, token, err := r.SessionUC.CreateSession(
		request.UserName,
		request.Password,
		clientIP,
		userAgent,
	)
	if err != nil {
		if errors.Is(err, sessions.ErrSessionAlreadyExists) {
			SessionConflictError(c)

			return
		}

		UnauthorizedError(c)

		return
	}

	// Set response headers
	c.Header(headerXAuthToken, token)
	c.Header(headerLocation, sessionBasePath+session.ID)

	// Convert to Redfish response format
	response, err := sessionToRedfishResponse(session)
	if err != nil {
		InternalServerError(c, fmt.Errorf("%s: %w", errMsgFailedBuildResponse, err))

		return
	}

	// Return session resource
	c.JSON(http.StatusCreated, response)
}

// GetRedfishV1SessionServiceSessionsSessionId handles GET /redfish/v1/SessionService/Sessions/{SessionId}.
//
//nolint:revive // Method name must match OpenAPI-generated interface
func (r *RedfishServer) GetRedfishV1SessionServiceSessionsSessionId(c *gin.Context, sessionId string) {
	SetRedfishHeaders(c)

	if sessionId == "" {
		BadRequestError(c, errMsgSessionIDRequired)

		return
	}

	session, err := r.SessionUC.GetSession(sessionId)
	if err != nil {
		if errors.Is(err, sessions.ErrSessionNotFound) || errors.Is(err, sessions.ErrSessionExpired) {
			NotFoundError(c, "Session", sessionId)

			return
		}

		InternalServerError(c, fmt.Errorf("%s: %w", errMsgFailedRetrieveSession, err))

		return
	}

	// Convert to Redfish response format
	response, err := sessionToRedfishResponse(session)
	if err != nil {
		InternalServerError(c, fmt.Errorf("%s: %w", errMsgFailedBuildResponse, err))

		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteRedfishV1SessionServiceSessionsSessionId handles DELETE /redfish/v1/SessionService/Sessions/{SessionId}.
//
//nolint:revive // Method name must match OpenAPI-generated interface
func (r *RedfishServer) DeleteRedfishV1SessionServiceSessionsSessionId(c *gin.Context, sessionId string) {
	SetRedfishHeaders(c)

	if sessionId == "" {
		BadRequestError(c, errMsgSessionIDRequired)

		return
	}

	err := r.SessionUC.DeleteSession(sessionId)
	if err != nil {
		if errors.Is(err, sessions.ErrSessionNotFound) {
			NotFoundError(c, "Session", sessionId)

			return
		}

		InternalServerError(c, fmt.Errorf("%s: %w", errMsgFailedDeleteSession, err))

		return
	}

	c.Status(http.StatusNoContent)
}

// PatchRedfishV1SessionService handles PATCH /redfish/v1/SessionService
func (r *RedfishServer) PatchRedfishV1SessionService(c *gin.Context) {
	SetRedfishHeaders(c)

	var req map[string]interface{}
	if err := c.BindJSON(&req); err != nil {
		BadRequestError(c, fmt.Sprintf("%s: %v", errMsgInvalidRequestBody, err))

		return
	}

	// For now, PATCH returns the current state without modifications.
	// Future: Implement PATCH logic to update configurable properties.
	response, err := r.buildSessionServiceResponse()
	if err != nil {
		InternalServerError(c, fmt.Errorf("%s: %w", errMsgFailedBuildServiceResp, err))

		return
	}

	c.JSON(http.StatusOK, response)
}

// PutRedfishV1SessionService handles PUT /redfish/v1/SessionService
func (r *RedfishServer) PutRedfishV1SessionService(c *gin.Context) {
	SetRedfishHeaders(c)

	var req map[string]interface{}
	if err := c.BindJSON(&req); err != nil {
		BadRequestError(c, fmt.Sprintf("%s: %v", errMsgInvalidRequestBody, err))

		return
	}

	// For now, PUT returns the current state without modifications.
	// Future: Implement PUT logic to replace the resource.
	response, err := r.buildSessionServiceResponse()
	if err != nil {
		InternalServerError(c, fmt.Errorf("%s: %w", errMsgFailedBuildServiceResp, err))

		return
	}

	c.JSON(http.StatusOK, response)
}
