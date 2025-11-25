package v1

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
)

// resetMetadataState safely resets the global metadata state for test isolation.
// This must be called within a synchronized context to avoid race conditions.
func resetMetadataState() {
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	metadataXML = ""
	metadataLoaded = false
}

// setupMetadataTestRouter creates and configures a test router with the metadata endpoint.
func setupMetadataTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	server := &RedfishServer{}
	router.GET("/redfish/v1/$metadata", server.GetRedfishV1Metadata)

	return router
}

func TestGetRedfishV1MetadataReturnsODataXML(t *testing.T) {
	t.Parallel()

	// Reset global state for test isolation
	t.Cleanup(func() {
		resetMetadataState()
	})

	gin.SetMode(gin.TestMode)

	// Setup
	router := setupMetadataTestRouter()

	// Execute request
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert status and headers
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/xml", w.Header().Get("Content-Type"))
	assert.Equal(t, "4.0", w.Header().Get("OData-Version"))

	body := w.Body.String()

	// Verify valid XML by parsing it (if body is not empty)
	if body != "" {
		assert.NoError(t, xml.Unmarshal([]byte(body), new(interface{})), "response should be valid XML")

		// Assert XML declaration and root element
		assert.True(t, strings.HasPrefix(body, `<?xml version="1.0" encoding="UTF-8"?>`))
		assert.Contains(t, body, `<edmx:Edmx`)
		assert.Contains(t, body, `xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx"`)
		assert.Contains(t, body, `Version="4.0"`)

		// Assert required DMTF Redfish schema references (auto-discovered from YAML files)
		requiredSchemas := []string{
			"ActionInfo_v1.xml",
			"ComputerSystemCollection_v1.xml",
			"ComputerSystem_v1.xml",
			"Message_v1.xml",
			"ResolutionStep_v1.xml",
			"Resource_v1.xml",
			"ServiceRoot_v1.xml",
		}
		for _, schema := range requiredSchemas {
			assert.Contains(t, body, schema, "metadata should reference %s schema", schema)
		}

		// Assert EntityContainer structure
		assert.Contains(t, body, `<edmx:DataServices>`)
		assert.Contains(t, body, `<Schema`)
		assert.Contains(t, body, `<EntityContainer Name="Service"`)
		assert.Contains(t, body, `Extends="ServiceRoot.v1_19_0.ServiceContainer"`)

		// Verify substantial content
		assert.Greater(t, len(body), 1000, "metadata should contain substantial content")
	}
}

func TestGetRedfishV1MetadataServeValidResponse(t *testing.T) {
	t.Parallel()

	// Reset global state for test isolation
	t.Cleanup(func() {
		resetMetadataState()
	})

	gin.SetMode(gin.TestMode)

	router := setupMetadataTestRouter()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify it's valid XML if response body is not empty
	body := w.Body.String()
	if body != "" {
		var doc interface{}

		err := xml.Unmarshal([]byte(body), &doc)
		assert.NoError(t, err, "response should be valid XML")
	}
}

func TestGetRedfishV1MetadataConcurrentRequests(t *testing.T) {
	t.Parallel()

	// Reset global state for test isolation
	t.Cleanup(func() {
		resetMetadataState()
	})

	gin.SetMode(gin.TestMode)

	router := setupMetadataTestRouter()

	// Execute multiple concurrent requests
	const numRequests = 10

	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			results <- w.Code
		}()
	}

	// Verify all requests succeeded
	for i := 0; i < numRequests; i++ {
		assert.Equal(t, http.StatusOK, <-results)
	}
}

// TestLoadMetadata tests the metadata loading behavior through the public endpoint.
// Since loadMetadata is internal, we test it indirectly through GetRedfishV1Metadata.
func TestLoadMetadata(t *testing.T) {
	t.Parallel()

	t.Run("metadata endpoint loads and caches metadata", func(t *testing.T) {
		t.Parallel()
		resetMetadataState()

		gin.SetMode(gin.TestMode)

		// Setup router
		router := setupMetadataTestRouter()

		// First request
		req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		// Second request - should be identical (cached)
		req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		// Both should have same status
		assert.Equal(t, w1.Code, w2.Code)

		// Both should have same response body (caching works)
		assert.Equal(t, w1.Body.String(), w2.Body.String())
	})

	t.Run("metadata endpoint sets correct headers", func(t *testing.T) {
		t.Parallel()
		resetMetadataState()

		gin.SetMode(gin.TestMode)

		router := setupMetadataTestRouter()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify headers are set correctly
		assert.Equal(t, "application/xml", w.Header().Get("Content-Type"))
		assert.Equal(t, "4.0", w.Header().Get("OData-Version"))
	})
}

// TestValidateMetadataXML tests XML validation through the endpoint.
// Direct validation testing is not possible since validateMetadataXML is private.
// Instead, we verify that the endpoint returns valid XML responses.
func TestValidateMetadataXML(t *testing.T) {
	t.Parallel()

	t.Run("endpoint returns valid XML response", func(t *testing.T) {
		t.Parallel()
		resetMetadataState()

		gin.SetMode(gin.TestMode)

		router := setupMetadataTestRouter()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		body := w.Body.String()

		// Verify response is valid XML
		if body != "" {
			var doc interface{}

			err := xml.Unmarshal([]byte(body), &doc)
			assert.NoError(t, err, "response should be valid XML")
		}

		// Verify status
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestLoadMetadataIntegration tests the metadata endpoint integration.
func TestLoadMetadataIntegration(t *testing.T) {
	t.Parallel()

	t.Run("metadata endpoint returns consistent results", func(t *testing.T) {
		t.Parallel()
		resetMetadataState()

		gin.SetMode(gin.TestMode)

		router := setupMetadataTestRouter()

		// Make multiple requests
		responses := make([]string, 3)

		for i := 0; i < 3; i++ {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			responses[i] = w.Body.String()
		}

		// All responses should be identical (caching works)
		assert.Equal(t, responses[0], responses[1])
		assert.Equal(t, responses[1], responses[2])
	})

	t.Run("metadata endpoint response has proper structure", func(t *testing.T) {
		t.Parallel()
		resetMetadataState()

		gin.SetMode(gin.TestMode)

		router := setupMetadataTestRouter()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()

		// If metadata is loaded, verify structure
		if body != "" {
			// Should be valid XML
			var doc interface{}

			err := xml.Unmarshal([]byte(body), &doc)
			assert.NoError(t, err, "response should be valid XML")

			// Verify headers
			assert.Equal(t, "application/xml", w.Header().Get("Content-Type"))
			assert.Equal(t, "4.0", w.Header().Get("OData-Version"))
		}
	})
}

// TestGetRedfishV1ServiceRoot tests the GET /redfish/v1 service root endpoint
func TestGetRedfishV1ServiceRoot(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	// Setup router
	router := gin.New()
	server := &RedfishServer{}
	router.GET("/redfish/v1", server.GetRedfishV1)

	// Execute request
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1", http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert status code
	assert.Equal(t, http.StatusOK, w.Code, "should return 200 OK")

	// Assert headers
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, "4.0", w.Header().Get("OData-Version"))

	// Parse response
	var response map[string]interface{}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err, "response should be valid JSON")

	// Assert OData metadata properties
	assert.Equal(t, "/redfish/v1/$metadata#ServiceRoot.ServiceRoot", response["@odata.context"])
	assert.Equal(t, "/redfish/v1", response["@odata.id"])
	assert.Equal(t, "#ServiceRoot.v1_19_0.ServiceRoot", response["@odata.type"])

	// Assert required ServiceRoot properties
	assert.Equal(t, "RootService", response["Id"])
	assert.Equal(t, "Root Service", response["Name"])
	assert.Equal(t, "1.19.0", response["RedfishVersion"])

	// Assert UUID is present and valid format
	uuidVal, exists := response["UUID"]
	assert.True(t, exists, "UUID should be present")
	assert.NotEmpty(t, uuidVal, "UUID should not be empty")

	// Validate UUID format (should be a valid UUID string)
	uuidStr, ok := uuidVal.(string)
	assert.True(t, ok, "UUID should be a string")
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, uuidStr, "UUID should be valid UUID format")

	// Assert Systems link
	systems, exists := response["Systems"].(map[string]interface{})
	assert.True(t, exists, "Systems should be present")
	assert.Equal(t, "/redfish/v1/Systems", systems["@odata.id"])
}

// TestGetRedfishV1ServiceRootUUIDConsistency tests UUID is consistent across multiple requests
func TestGetRedfishV1ServiceRootUUIDConsistency(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	// Setup router
	router := gin.New()
	server := &RedfishServer{}
	router.GET("/redfish/v1", server.GetRedfishV1)

	// Make multiple requests
	uuids := make([]string, 3)

	for i := 0; i < 3; i++ {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1", http.NoBody)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response map[string]interface{}

		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		uuid, ok := response["UUID"].(string)
		require.True(t, ok, "UUID should be a string")

		uuids[i] = uuid
	}

	// All UUIDs should be identical (deterministic generation)
	assert.Equal(t, uuids[0], uuids[1], "UUID should be consistent across requests")
	assert.Equal(t, uuids[1], uuids[2], "UUID should be consistent across requests")
}

// TestGetRedfishV1ServiceRootNoAuthentication validates endpoint is accessible without authentication
func TestGetRedfishV1ServiceRootNoAuthentication(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	// Setup router without authentication middleware
	router := gin.New()
	server := &RedfishServer{}
	router.GET("/redfish/v1", server.GetRedfishV1)

	// Execute request without Authorization header
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1", http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed without authentication
	assert.Equal(t, http.StatusOK, w.Code, "service root should be accessible without authentication")
}

// TestGetRedfishV1ServiceRootRedfishCompliance validates Redfish specification compliance
func TestGetRedfishV1ServiceRootRedfishCompliance(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	// Setup router
	router := gin.New()
	server := &RedfishServer{}
	router.GET("/redfish/v1", server.GetRedfishV1)

	// Execute request
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1", http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Parse response
	var response map[string]interface{}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Validate Redfish mandatory properties per ServiceRoot.v1_19_0 schema
	mandatoryProperties := []string{
		"@odata.context",
		"@odata.id",
		"@odata.type",
		"Id",
		"Name",
	}

	for _, prop := range mandatoryProperties {
		_, exists := response[prop]
		assert.True(t, exists, "mandatory property %s should be present", prop)
	}

	// Validate recommended properties are present
	recommendedProperties := []string{
		"RedfishVersion",
		"UUID",
		"Systems",
	}

	for _, prop := range recommendedProperties {
		_, exists := response[prop]
		assert.True(t, exists, "recommended property %s should be present", prop)
	}
}

// TestGetRedfishV1ServiceRootConcurrentRequests validates thread safety
func TestGetRedfishV1ServiceRootConcurrentRequests(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	// Setup router
	router := gin.New()
	server := &RedfishServer{}
	router.GET("/redfish/v1", server.GetRedfishV1)

	// Execute multiple concurrent requests
	const numRequests = 20

	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1", http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			results <- w.Code
		}()
	}

	// Verify all requests succeeded
	for i := 0; i < numRequests; i++ {
		assert.Equal(t, http.StatusOK, <-results)
	}
}

// TestGetRedfishV1ServiceRootResponseStructure validates the complete response structure
func TestGetRedfishV1ServiceRootResponseStructure(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	// Setup router
	router := gin.New()
	server := &RedfishServer{}
	router.GET("/redfish/v1", server.GetRedfishV1)

	// Execute request
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1", http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Parse into the generated type to ensure type compatibility
	var serviceRoot generated.ServiceRootServiceRoot

	err = json.Unmarshal(w.Body.Bytes(), &serviceRoot)
	require.NoError(t, err, "response should unmarshal into ServiceRootServiceRoot type")

	// Validate all pointer fields are properly set
	assert.NotNil(t, serviceRoot.OdataContext)
	assert.NotNil(t, serviceRoot.OdataId)
	assert.NotNil(t, serviceRoot.OdataType)
	assert.NotNil(t, serviceRoot.RedfishVersion)
	assert.NotNil(t, serviceRoot.UUID)
	assert.NotNil(t, serviceRoot.Systems)
	assert.NotNil(t, serviceRoot.Systems.OdataId)

	// Validate values
	assert.Equal(t, "/redfish/v1/$metadata#ServiceRoot.ServiceRoot", *serviceRoot.OdataContext)
	assert.Equal(t, "/redfish/v1", *serviceRoot.OdataId)
	assert.Equal(t, "#ServiceRoot.v1_19_0.ServiceRoot", *serviceRoot.OdataType)
	assert.Equal(t, "RootService", serviceRoot.Id)
	assert.Equal(t, "Root Service", serviceRoot.Name)
	assert.Equal(t, "1.19.0", *serviceRoot.RedfishVersion)
	assert.Equal(t, "/redfish/v1/Systems", *serviceRoot.Systems.OdataId)
}
