package openapi

// NoContentResponse models endpoints that intentionally return HTTP 204.
type NoContentResponse struct{}

// ErrorResponse models error responses with error and message fields.
type ErrorResponse struct {
	Error   string `json:"error,omitempty" example:"invalid credentials"`
	Message string `json:"message,omitempty" example:"Incorrect username or password"`
}
