package v1

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOData_BindAndValidate(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		queryString string
		wantErr     bool
		errContains string
		wantTop     int
		wantSkip    int
		wantCount   bool
	}{
		{
			name:        "valid parameters with default top",
			queryString: "?$skip=10&$count=true",
			wantErr:     false,
			wantTop:     25, // default value
			wantSkip:    10,
			wantCount:   true,
		},
		{
			name:        "valid parameters with custom top",
			queryString: "?$top=100&$skip=50&$count=false",
			wantErr:     false,
			wantTop:     100,
			wantSkip:    50,
			wantCount:   false,
		},
		{
			name:        "empty query parameters",
			queryString: "",
			wantErr:     false,
			wantTop:     25, // default value
			wantSkip:    0,
			wantCount:   false,
		},
		{
			name:        "integer overflow on top parameter",
			queryString: "?$top=18628037784217768768463850568257440918905312",
			wantErr:     true,
			errContains: "exceeds maximum allowed range",
		},
		{
			name:        "integer overflow on skip parameter",
			queryString: "?$skip=28866823837974031616722743261592341250719431",
			wantErr:     true,
			errContains: "exceeds maximum allowed range",
		},
		{
			name:        "integer overflow on count parameter (not integer)",
			queryString: "?$count=18628037784217768768463850568257440918905312",
			wantErr:     true,
			errContains: "must be a boolean value",
		},
		{
			name:        "negative top parameter",
			queryString: "?$top=-10",
			wantErr:     true,
			errContains: "must be non-negative",
		},
		{
			name:        "negative skip parameter",
			queryString: "?$skip=-5",
			wantErr:     true,
			errContains: "must be non-negative",
		},
		{
			name:        "top exceeds maximum allowed",
			queryString: fmt.Sprintf("?$top=%d", MaxPageSize+1),
			wantErr:     true,
			errContains: "exceeds maximum allowed value",
		},
		{
			name:        "skip exceeds maximum allowed",
			queryString: fmt.Sprintf("?$skip=%d", MaxSkipValue+1),
			wantErr:     true,
			errContains: "exceeds maximum allowed value",
		},
		{
			name:        "top exceeds int32 max",
			queryString: fmt.Sprintf("?$top=%d", int64(math.MaxInt32)+1),
			wantErr:     true,
			errContains: "exceeds maximum allowed range",
		},
		{
			name:        "skip exceeds int32 max",
			queryString: fmt.Sprintf("?$skip=%d", int64(math.MaxInt32)+1),
			wantErr:     true,
			errContains: "exceeds maximum allowed range",
		},
		{
			name:        "non-numeric top parameter",
			queryString: "?$top=abc",
			wantErr:     true,
			errContains: "must be a valid integer",
		},
		{
			name:        "non-numeric skip parameter",
			queryString: "?$skip=xyz",
			wantErr:     true,
			errContains: "must be a valid integer",
		},
		{
			name:        "maximum allowed top value",
			queryString: fmt.Sprintf("?$top=%d", MaxPageSize),
			wantErr:     false,
			wantTop:     MaxPageSize,
			wantSkip:    0,
			wantCount:   false,
		},
		{
			name:        "maximum allowed skip value",
			queryString: fmt.Sprintf("?$skip=%d", MaxSkipValue),
			wantErr:     false,
			wantTop:     25,
			wantSkip:    MaxSkipValue,
			wantCount:   false,
		},
		{
			name:        "boundary test - zero values",
			queryString: "?$top=0&$skip=0",
			wantErr:     false,
			wantTop:     0,
			wantSkip:    0,
			wantCount:   false,
		},
		{
			name:        "float top parameter (invalid)",
			queryString: "?$top=10.5",
			wantErr:     true,
			errContains: "must be a valid integer",
		},
		{
			name:        "scientific notation (large number)",
			queryString: "?$top=1e20",
			wantErr:     true,
			errContains: "must be a valid integer",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create test request and context
			req := httptest.NewRequest(http.MethodGet, "/test"+tt.queryString, http.NoBody)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Test the BindAndValidate method
			var odata OData

			err := odata.BindAndValidate(c)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantTop, odata.Top)
				assert.Equal(t, tt.wantSkip, odata.Skip)
				assert.Equal(t, tt.wantCount, odata.Count)
			}
		})
	}
}

func TestParseAndValidateInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		value      string
		maxAllowed int
		want       int
		wantErr    bool
	}{
		{
			name:       "valid positive integer",
			value:      "100",
			maxAllowed: 1000,
			want:       100,
			wantErr:    false,
		},
		{
			name:       "zero value",
			value:      "0",
			maxAllowed: 1000,
			want:       0,
			wantErr:    false,
		},
		{
			name:       "negative value",
			value:      "-10",
			maxAllowed: 1000,
			wantErr:    true,
		},
		{
			name:       "exceeds max allowed",
			value:      "2000",
			maxAllowed: 1000,
			wantErr:    true,
		},
		{
			name:       "exceeds int32 max",
			value:      fmt.Sprintf("%d", int64(math.MaxInt32)+1),
			maxAllowed: math.MaxInt32,
			wantErr:    true,
		},
		{
			name:       "overflow - number too large",
			value:      "999999999999999999999999999999",
			maxAllowed: 1000,
			wantErr:    true,
		},
		{
			name:       "non-numeric string",
			value:      "abc",
			maxAllowed: 1000,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseAndValidateInt(tt.value, tt.maxAllowed)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
