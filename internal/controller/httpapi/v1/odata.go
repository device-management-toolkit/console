package v1

import (
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	// MaxPageSize is the maximum allowed value for $top parameter.
	MaxPageSize = 500
	// MaxSkipValue is the maximum allowed value for $skip parameter.
	MaxSkipValue = 100000
	// DefaultPageSize is the default value for $top when not specified.
	DefaultPageSize = 25
	// DecimalBase is the base for decimal integer parsing.
	DecimalBase = 10
	// BitSize64 is the bit size for 64-bit integer parsing.
	BitSize64 = 64
)

var (
	// ErrInvalidInteger is returned when a query parameter is not a valid integer.
	ErrInvalidInteger = errors.New("parameter must be a valid integer")
	// ErrExceedsMaxRange is returned when a query parameter exceeds the maximum allowed range.
	ErrExceedsMaxRange = errors.New("parameter value exceeds maximum allowed range")
	// ErrNegativeValue is returned when a query parameter is negative.
	ErrNegativeValue = errors.New("parameter must be non-negative")
	// ErrInvalidBoolean is returned when a boolean query parameter is not a valid boolean.
	ErrInvalidBoolean = errors.New("parameter must be a boolean value")
)

// ValidationError represents a query parameter validation error with additional context.
type ValidationError struct {
	Max int
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("parameter exceeds maximum allowed value of %d", e.Max)
}

type OData struct {
	Top   int  `form:"$top,default=25"`
	Skip  int  `form:"$skip"`
	Count bool `form:"$count"`
}

// BindAndValidate binds query parameters and validates them against overflow and range limits.
func (o *OData) BindAndValidate(c *gin.Context) error {
	// Validate $top parameter
	if topStr := c.Query("$top"); topStr != "" {
		topVal, err := parseAndValidateInt(topStr, MaxPageSize)
		if err != nil {
			return fmt.Errorf("invalid $top parameter: %w", err)
		}

		o.Top = topVal
	} else {
		o.Top = DefaultPageSize
	}

	// Validate $skip parameter
	if skipStr := c.Query("$skip"); skipStr != "" {
		skipVal, err := parseAndValidateInt(skipStr, MaxSkipValue)
		if err != nil {
			return fmt.Errorf("invalid $skip parameter: %w", err)
		}

		o.Skip = skipVal
	}

	// Validate $count parameter (boolean)
	if countStr := c.Query("$count"); countStr != "" {
		countVal, err := strconv.ParseBool(countStr)
		if err != nil {
			return fmt.Errorf("invalid $count parameter: %w", ErrInvalidBoolean)
		}

		o.Count = countVal
	}

	return nil
}

// parseAndValidateInt safely parses a string to int and validates range.
func parseAndValidateInt(value string, maxAllowed int) (int, error) {
	// First try to parse as int64 to detect overflow beyond int range
	val64, err := strconv.ParseInt(value, DecimalBase, BitSize64)
	if err != nil {
		// Check if it's a range error (number too large)
		var numErr *strconv.NumError
		if ok := errors.As(err, &numErr); ok && errors.Is(numErr.Err, strconv.ErrRange) {
			return 0, ErrExceedsMaxRange
		}

		return 0, ErrInvalidInteger
	}

	// Check if value is negative
	if val64 < 0 {
		return 0, ErrNegativeValue
	}

	// Check if value exceeds int max (for 32-bit systems compatibility)
	if val64 > math.MaxInt32 {
		return 0, ErrExceedsMaxRange
	}

	// Check against application-specific maximum
	if val64 > int64(maxAllowed) {
		return 0, &ValidationError{Max: maxAllowed}
	}

	return int(val64), nil
}
