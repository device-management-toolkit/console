package mcp

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ErrInvalidGUID is returned when a tool receives a GUID that is not a valid UUID.
var ErrInvalidGUID = errors.New("guid must be a valid UUID")

// ErrUnknownPowerAction is returned when power_action receives an action name
// that is not in powerActionNames.
var ErrUnknownPowerAction = errors.New("unknown power action")

// validateGUID checks that the supplied device GUID is a syntactically valid
// UUID. The REST layer relies on the router to constrain the path parameter;
// MCP tools have no such router, so validation is explicit here.
func validateGUID(guid string) error {
	if _, err := uuid.Parse(guid); err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidGUID, guid)
	}

	return nil
}

// errorResult adapts a Go error into an MCP tool-call error result. The SDK
// ignores structured output when the result is flagged as an error, so callers
// return the zero value of their output type alongside this result.
func errorResult(err error) *sdk.CallToolResult {
	return &sdk.CallToolResult{
		IsError: true,
		Content: []sdk.Content{&sdk.TextContent{Text: err.Error()}},
	}
}
