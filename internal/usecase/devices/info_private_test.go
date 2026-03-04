package devices

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type parseCIMTest struct {
	name     string
	input    interface{}
	expected dto.CIMResponse
}

func TestParseCIMResponse(t *testing.T) {
	t.Parallel()

	// Create a minimal UseCase instance for testing
	useCase := &UseCase{}

	tests := []parseCIMTest{
		{
			name:     "nil input",
			input:    nil,
			expected: dto.CIMResponse{},
		},
		{
			name:     "non-map input",
			input:    "not a map",
			expected: dto.CIMResponse{},
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: dto.CIMResponse{},
		},
		{
			name: "map with response field only",
			input: map[string]interface{}{
				"response": "test response data",
			},
			expected: dto.CIMResponse{
				Response: "test response data",
			},
		},
		{
			name: "map with status field as int",
			input: map[string]interface{}{
				"status": 200,
			},
			expected: dto.CIMResponse{
				Status: 200,
			},
		},
		{
			name: "map with status field as non-int",
			input: map[string]interface{}{
				"status": "not an int",
			},
			expected: dto.CIMResponse{}, // status should remain 0
		},
		{
			name: "map with responses field as []interface{}",
			input: map[string]interface{}{
				"responses": []interface{}{"response1", "response2", 42},
			},
			expected: dto.CIMResponse{
				Responses: []interface{}{"response1", "response2", 42},
			},
		},
		{
			name: "map with responses field as typed slice - []string",
			input: map[string]interface{}{
				"responses": []string{"response1", "response2"},
			},
			expected: dto.CIMResponse{
				Responses: []interface{}{"response1", "response2"},
			},
		},
		{
			name: "map with responses field as typed slice - []int",
			input: map[string]interface{}{
				"responses": []int{1, 2, 3},
			},
			expected: dto.CIMResponse{
				Responses: []interface{}{1, 2, 3},
			},
		},
		{
			name: "map with responses field as empty typed slice",
			input: map[string]interface{}{
				"responses": []string{},
			},
			expected: dto.CIMResponse{
				Responses: []interface{}{},
			},
		},
		{
			name: "map with responses field as nil",
			input: map[string]interface{}{
				"responses": nil,
			},
			expected: dto.CIMResponse{}, // responses should remain nil
		},
		{
			name: "map with responses field as non-slice",
			input: map[string]interface{}{
				"responses": "not a slice",
			},
			expected: dto.CIMResponse{}, // responses should remain nil
		},
		{
			name: "map with responses field as non-slice number",
			input: map[string]interface{}{
				"responses": 123,
			},
			expected: dto.CIMResponse{}, // responses should remain nil
		},
		{
			name: "complete map with all valid fields",
			input: map[string]interface{}{
				"response":  "complete response",
				"responses": []interface{}{"item1", "item2"},
				"status":    404,
			},
			expected: dto.CIMResponse{
				Response:  "complete response",
				Responses: []interface{}{"item1", "item2"},
				Status:    404,
			},
		},
		{
			name: "complete map with typed slice responses",
			input: map[string]interface{}{
				"response":  42,
				"responses": []string{"typed1", "typed2"},
				"status":    201,
			},
			expected: dto.CIMResponse{
				Response:  42,
				Responses: []interface{}{"typed1", "typed2"},
				Status:    201,
			},
		},
		{
			name: "map with mixed valid and invalid fields",
			input: map[string]interface{}{
				"response":  "valid response",
				"responses": "invalid responses",
				"status":    "invalid status",
			},
			expected: dto.CIMResponse{
				Response: "valid response",
				// Responses should remain nil due to invalid type
				// Status should remain 0 due to invalid type
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := useCase.parseCIMResponse(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}
