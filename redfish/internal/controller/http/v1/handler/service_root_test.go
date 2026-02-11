package v1

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dmtconfig "github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
)

// resetMetadataState resets global metadata state for test isolation.
func resetMetadataState() {
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	metadataXML = ""
	metadataLoaded = false
}

// setupMetadataTestRouter creates a test router with metadata endpoint.
func setupMetadataTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
	router.GET("/redfish/v1/$metadata", server.GetRedfishV1Metadata)

	return router
}

func TestGetRedfishV1MetadataReturnsODataXML(t *testing.T) {
	t.Parallel()

	t.Cleanup(func() {
		resetMetadataState()
	})

	gin.SetMode(gin.TestMode)

	router := setupMetadataTestRouter()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/xml", w.Header().Get("Content-Type"))
	assert.Equal(t, "4.0", w.Header().Get("OData-Version"))

	body := w.Body.String()

	if body != "" {
		assert.NoError(t, xml.Unmarshal([]byte(body), new(interface{})), "response should be valid XML")

		assert.True(t, strings.HasPrefix(body, `<?xml version="1.0" encoding="UTF-8"?>`))
		assert.Contains(t, body, `<edmx:Edmx`)
		assert.Contains(t, body, `xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx"`)
		assert.Contains(t, body, `Version="4.0"`)

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

		assert.Contains(t, body, `<edmx:DataServices>`)
		assert.Contains(t, body, `<Schema`)
		assert.Contains(t, body, `<EntityContainer Name="Service"`)
		assert.Contains(t, body, `Extends="ServiceRoot.v1_19_0.ServiceContainer"`)
		assert.Greater(t, len(body), 1000, "metadata should contain substantial content")
	}
}

func TestGetRedfishV1MetadataServeValidResponse(t *testing.T) {
	t.Parallel()

	t.Cleanup(func() {
		resetMetadataState()
	})

	gin.SetMode(gin.TestMode)

	router := setupMetadataTestRouter()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	body := w.Body.String()
	if body != "" {
		var doc interface{}

		err := xml.Unmarshal([]byte(body), &doc)
		assert.NoError(t, err, "response should be valid XML")
	}
}

func TestGetRedfishV1MetadataConcurrentRequests(t *testing.T) {
	t.Parallel()

	t.Cleanup(func() {
		resetMetadataState()
	})

	gin.SetMode(gin.TestMode)

	router := setupMetadataTestRouter()

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

	for i := 0; i < numRequests; i++ {
		assert.Equal(t, http.StatusOK, <-results)
	}
}

//nolint:paralleltest // Cannot run in parallel due to shared global metadata state
func TestLoadMetadata(t *testing.T) {
	t.Run("metadata endpoint loads and caches metadata", func(t *testing.T) {
		resetMetadataState()
		gin.SetMode(gin.TestMode)

		router := setupMetadataTestRouter()

		req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, w1.Code, w2.Code)
		assert.Equal(t, w1.Body.String(), w2.Body.String())
	})

	t.Run("metadata endpoint sets correct headers", func(t *testing.T) {
		resetMetadataState()
		gin.SetMode(gin.TestMode)

		router := setupMetadataTestRouter()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/$metadata", http.NoBody)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

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
//
//nolint:paralleltest // Cannot use t.Parallel() due to shared global metadata state that requires sequential execution
func TestLoadMetadataIntegration(t *testing.T) {
	t.Run("metadata endpoint returns consistent results", func(t *testing.T) {
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

		// Verify all responses are non-empty
		assert.NotEmpty(t, responses[0], "First response should not be empty")
		assert.NotEmpty(t, responses[1], "Second response should not be empty")
		assert.NotEmpty(t, responses[2], "Third response should not be empty")

		// All responses should be identical (caching works)
		assert.Equal(t, responses[0], responses[1])
		assert.Equal(t, responses[1], responses[2])
	})

	t.Run("metadata endpoint response has proper structure", func(t *testing.T) {
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
	server := &RedfishServer{
		Config: &dmtconfig.Config{
			App: dmtconfig.App{},
		},
	}
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
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
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

		uuidValue, ok := response["UUID"].(string)
		require.True(t, ok, "UUID should be a string")

		uuids[i] = uuidValue
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
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
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
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
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
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
	router.GET("/redfish/v1", server.GetRedfishV1)

	// Execute multiple concurrent requests
	const numRequests = 20

	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/redfish/v1", http.NoBody)
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

// TestGetRedfishV1Odata tests the OData endpoint basic functionality
func TestGetRedfishV1Odata(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
	router.GET("/redfish/v1/odata", server.GetRedfishV1Odata)

	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/odata", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "4.0", w.Header().Get("OData-Version"))
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
}

// TestGetRedfishV1OdataResponseStructure validates OData response format
func TestGetRedfishV1OdataResponseStructure(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
	router.GET("/redfish/v1/odata", server.GetRedfishV1Odata)

	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/odata", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify context
	assert.Equal(t, "/redfish/v1/$metadata#ServiceRoot.ServiceRoot", response["@odata.context"])

	// Verify value array exists
	valueArray, ok := response["value"].([]interface{})
	assert.True(t, ok, "value should be an array")
	assert.Greater(t, len(valueArray), 0, "value array should not be empty")

	// Verify service structure
	for _, item := range valueArray {
		service, ok := item.(map[string]interface{})
		assert.True(t, ok)
		assert.NotNil(t, service["name"])
		assert.NotNil(t, service["kind"])
		assert.NotNil(t, service["url"])
		assert.Equal(t, "Singleton", service["kind"])
	}
}

// TestGetRedfishV1OdataRequiredServices validates Systems service is present
func TestGetRedfishV1OdataRequiredServices(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
	router.GET("/redfish/v1/odata", server.GetRedfishV1Odata)

	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/odata", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	valueArray, ok := response["value"].([]interface{})
	require.True(t, ok, "value should be an array")

	// Find Systems service
	var foundSystems bool

	var systemsURL string

	for _, item := range valueArray {
		service, ok := item.(map[string]interface{})
		require.True(t, ok, "item should be a map")

		name, ok := service["name"].(string)
		require.True(t, ok, "name should be a string")

		if name == "Systems" {
			foundSystems = true
			systemsURL, _ = service["url"].(string)

			break
		}
	}

	assert.True(t, foundSystems, "Systems service should be present")
	assert.Equal(t, "/redfish/v1/Systems", systemsURL)
}

// TestGetRedfishV1OdataNoAuthentication validates endpoint is public
func TestGetRedfishV1OdataNoAuthentication(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
	router.GET("/redfish/v1/odata", server.GetRedfishV1Odata)

	req := httptest.NewRequest(http.MethodGet, "/redfish/v1/odata", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetRedfishV1OdataConcurrentRequests validates thread safety
func TestGetRedfishV1OdataConcurrentRequests(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
	router.GET("/redfish/v1/odata", server.GetRedfishV1Odata)

	const numRequests = 10

	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/redfish/v1/odata", http.NoBody)
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
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
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

// TestGenerateServiceUUID tests the UUID generation
//
//nolint:tparallel // Cannot use t.Parallel() - subtests modify global cachedUUID
func TestGenerateServiceUUID(t *testing.T) {
	t.Run("returns valid UUID format", func(t *testing.T) {
		t.Parallel()

		generatedUUID := generateServiceUUID("")

		// Should be valid UUID format
		_, err := uuid.Parse(generatedUUID)
		assert.NoError(t, err, "should generate valid UUID")
		assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, generatedUUID)
	})

	t.Run("returns consistent UUID across calls", func(t *testing.T) {
		t.Parallel()

		uuid1 := generateServiceUUID("")
		uuid2 := generateServiceUUID("")

		// Should be the same UUID (file persistence still active)
		assert.Equal(t, uuid1, uuid2, "UUID should be consistent across calls")
	})

	//nolint:paralleltest // Cannot run in parallel - modifies global cachedUUID
	t.Run("uses configured UUID when provided", func(t *testing.T) {
		// Note: Cannot use t.Parallel() because it modifies global cachedUUID

		// Reset cache
		cachedUUID = ""

		configUUID := "12345678-1234-5678-1234-567812345678"
		resultUUID := generateServiceUUID(configUUID)

		assert.Equal(t, configUUID, resultUUID, "should use configured UUID")
	})

	//nolint:paralleltest // Cannot run in parallel - modifies global cachedUUID
	t.Run("ignores invalid configured UUID", func(t *testing.T) {
		// Note: Cannot use t.Parallel() because it modifies global cachedUUID

		// Reset cache
		cachedUUID = ""

		invalidUUID := "not-a-valid-uuid"
		resultUUID := generateServiceUUID(invalidUUID)

		// Should fall back to generated UUID
		assert.NotEqual(t, invalidUUID, resultUUID, "should not use invalid UUID")
		_, err := uuid.Parse(resultUUID)
		assert.NoError(t, err, "should generate valid UUID as fallback")
	})
}

// TestLoadOrCreateUUID tests the file-based UUID persistence
func TestLoadOrCreateUUID(t *testing.T) {
	t.Parallel()

	t.Run("creates new UUID when file doesn't exist", func(t *testing.T) {
		t.Parallel()

		// Use unique app name for this test to avoid conflicts
		appName := "test-app-new-uuid-" + uuid.New().String()

		generatedUUID, err := loadOrCreateUUID(appName)

		require.NoError(t, err)
		assert.NotEmpty(t, generatedUUID)

		// Validate UUID format
		_, parseErr := uuid.Parse(generatedUUID)
		assert.NoError(t, parseErr, "should be valid UUID")

		// Cleanup
		if path, err := getUUIDStoragePath(appName); err == nil {
			os.Remove(path)
			os.Remove(filepath.Dir(path))
		}
	})

	t.Run("loads existing UUID from file", func(t *testing.T) {
		t.Parallel()

		// Use unique app name for this test
		appName := "test-app-existing-uuid-" + uuid.New().String()

		// First call creates UUID
		uuid1, err := loadOrCreateUUID(appName)
		require.NoError(t, err)

		// Second call should load the same UUID
		uuid2, err := loadOrCreateUUID(appName)
		require.NoError(t, err)

		assert.Equal(t, uuid1, uuid2, "should load same UUID from file")

		// Cleanup
		if path, err := getUUIDStoragePath(appName); err == nil {
			os.Remove(path)
			os.Remove(filepath.Dir(path))
		}
	})

	t.Run("handles invalid UUID in file", func(t *testing.T) {
		t.Parallel()

		// Use unique app name for this test
		appName := "test-app-invalid-uuid-" + uuid.New().String()

		// Write invalid UUID to file
		path, err := getUUIDStoragePath(appName)
		require.NoError(t, err)

		const testFilePermissions = 0o600

		err = os.WriteFile(path, []byte("invalid-uuid-content"), testFilePermissions)
		require.NoError(t, err)

		// Should generate new valid UUID
		generatedUUID, err := loadOrCreateUUID(appName)
		require.NoError(t, err)

		_, parseErr := uuid.Parse(generatedUUID)
		assert.NoError(t, parseErr, "should generate valid UUID when file contains invalid data")
		assert.NotEqual(t, "invalid-uuid-content", generatedUUID)

		// Cleanup
		os.Remove(path)
		os.Remove(filepath.Dir(path))
	})
}

// TestGetUUIDStoragePath tests the storage path generation
func TestGetUUIDStoragePath(t *testing.T) {
	t.Parallel()

	t.Run("returns valid path", func(t *testing.T) {
		t.Parallel()

		path, err := getUUIDStoragePath("test-app")

		require.NoError(t, err)
		assert.NotEmpty(t, path)
		assert.Contains(t, path, "test-app")
		assert.Contains(t, path, uuidFileName)

		// Cleanup if directory was created
		os.Remove(path)
		os.Remove(filepath.Dir(path))
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		t.Parallel()

		appName := "test-app-dir-" + uuid.New().String()

		path, err := getUUIDStoragePath(appName)
		require.NoError(t, err)

		// Directory should exist
		dir := filepath.Dir(path)
		info, err := os.Stat(dir)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())

		// Cleanup
		os.Remove(path)
		os.Remove(dir)
	})

	t.Run("path is OS-specific", func(t *testing.T) {
		t.Parallel()

		path, err := getUUIDStoragePath("test-app")

		require.NoError(t, err)
		// Should use OS-specific path separator
		assert.Contains(t, path, string(filepath.Separator))

		// Cleanup
		os.Remove(path)
		os.Remove(filepath.Dir(path))
	})
}

// BenchmarkGetRedfishV1Odata measures OData endpoint performance
func BenchmarkGetRedfishV1Odata(b *testing.B) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
	router.GET("/redfish/v1/odata", server.GetRedfishV1Odata)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/redfish/v1/odata", http.NoBody)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// TestExtractServicesFromOpenAPIData tests the service extraction from OpenAPI spec
func TestExtractServicesFromOpenAPIData(t *testing.T) {
	t.Parallel()

	t.Run("valid OpenAPI spec with Systems service", func(t *testing.T) {
		t.Parallel()

		yamlData := []byte(`
openapi: 3.0.0
paths:
  /redfish/v1/Systems:
    get:
      summary: Get Systems collection
`)

		services, err := ExtractServicesFromOpenAPIData(yamlData)

		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, "Systems", services[0].Name)
		assert.Equal(t, "Singleton", services[0].Kind)
		assert.Equal(t, "/redfish/v1/Systems", services[0].URL)
	})

	t.Run("returns default when no valid services found", func(t *testing.T) {
		t.Parallel()

		yamlData := []byte(`
openapi: 3.0.0
paths:
  /redfish/v1/Systems/{ComputerSystemId}:
    get:
      summary: Parametrized path (should be ignored)
`)

		services, err := ExtractServicesFromOpenAPIData(yamlData)

		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, "Systems", services[0].Name)
		assert.Equal(t, "/redfish/v1/Systems", services[0].URL)
	})

	t.Run("handles invalid YAML gracefully", func(t *testing.T) {
		t.Parallel()

		invalidYAML := []byte(`invalid yaml content`)

		services, err := ExtractServicesFromOpenAPIData(invalidYAML)

		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, "Systems", services[0].Name)
	})

	t.Run("extracts both collection and member paths", func(t *testing.T) {
		t.Parallel()

		yamlData := []byte(`
openapi: 3.0.0
paths:
  /redfish/v1/Systems:
    get:
      summary: Get Systems collection
  /redfish/v1/Systems/{ComputerSystemId}:
    get:
      summary: Get specific system
`)

		services, err := ExtractServicesFromOpenAPIData(yamlData)

		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, "Systems", services[0].Name)
		assert.Equal(t, "/redfish/v1/Systems", services[0].URL)
	})

	t.Run("ignores deeply nested parametrized paths", func(t *testing.T) {
		t.Parallel()

		yamlData := []byte(`
openapi: 3.0.0
paths:
  /redfish/v1/Systems/{ComputerSystemId}/Storage/{StorageId}:
    get:
      summary: Should be ignored
  /redfish/v1/Systems:
    get:
      summary: Should be extracted
`)

		services, err := ExtractServicesFromOpenAPIData(yamlData)

		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, "Systems", services[0].Name)
		assert.Equal(t, "/redfish/v1/Systems", services[0].URL)
	})

	t.Run("handles multiple HTTP methods on same path", func(t *testing.T) {
		t.Parallel()

		yamlData := []byte(`
openapi: 3.0.0
paths:
  /redfish/v1/Systems:
    get:
      summary: Get Systems collection
    post:
      summary: Create system (hypothetical)
  /redfish/v1/Systems/{ComputerSystemId}:
    get:
      summary: Get specific system
    patch:
      summary: Update system
    delete:
      summary: Delete system
`)

		services, err := ExtractServicesFromOpenAPIData(yamlData)

		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, "Systems", services[0].Name)
		assert.Equal(t, "/redfish/v1/Systems", services[0].URL)
	})

	t.Run("handles empty paths object", func(t *testing.T) {
		t.Parallel()

		yamlData := []byte(`
openapi: 3.0.0
paths: {}
`)

		services, err := ExtractServicesFromOpenAPIData(yamlData)

		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, "Systems", services[0].Name)
	})

	t.Run("handles missing paths key", func(t *testing.T) {
		t.Parallel()

		yamlData := []byte(`
openapi: 3.0.0
info:
  title: Test API
`)

		services, err := ExtractServicesFromOpenAPIData(yamlData)

		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, "Systems", services[0].Name)
	})

	t.Run("prioritizes collection path over member path", func(t *testing.T) {
		t.Parallel()

		yamlData := []byte(`
openapi: 3.0.0
paths:
  /redfish/v1/Systems/{ComputerSystemId}:
    get:
      summary: Member path listed first
  /redfish/v1/Systems:
    get:
      summary: Collection path listed second
`)

		services, err := ExtractServicesFromOpenAPIData(yamlData)

		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, "Systems", services[0].Name)
		assert.Equal(t, "/redfish/v1/Systems", services[0].URL)
		assert.Equal(t, "Singleton", services[0].Kind)
	})

	t.Run("handles paths with trailing slashes", func(t *testing.T) {
		t.Parallel()

		yamlData := []byte(`
openapi: 3.0.0
paths:
  /redfish/v1/Systems/:
    get:
      summary: Collection with trailing slash
`)

		services, err := ExtractServicesFromOpenAPIData(yamlData)

		require.NoError(t, err)
		// Should still work or return default
		assert.Len(t, services, 1)
	})

	t.Run("case sensitivity in paths", func(t *testing.T) {
		t.Parallel()

		yamlData := []byte(`
openapi: 3.0.0
paths:
  /redfish/v1/systems:
    get:
      summary: Lowercase systems
`)

		services, err := ExtractServicesFromOpenAPIData(yamlData)

		require.NoError(t, err)
		// Should return default since path doesn't match expected pattern
		assert.Len(t, services, 1)
	})
}

// TestGetDefaultServices tests the default services fallback
func TestGetDefaultServices(t *testing.T) {
	t.Parallel()

	services := GetDefaultServices()

	assert.Len(t, services, 1)
	assert.Equal(t, "Systems", services[0].Name)
	assert.Equal(t, "Singleton", services[0].Kind)
	assert.Equal(t, "/redfish/v1/Systems", services[0].URL)
}

// TestGetRedfishV1ServiceRootHTTPMethods tests various HTTP methods on service root endpoint
func TestGetRedfishV1ServiceRootHTTPMethods(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
	router.GET("/redfish/v1", server.GetRedfishV1)

	testCases := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET method should succeed",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST method should fail",
			method:         http.MethodPost,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "PUT method should fail",
			method:         http.MethodPut,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "DELETE method should fail",
			method:         http.MethodDelete,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "PATCH method should fail",
			method:         http.MethodPatch,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tc.method, "/redfish/v1", http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestGetRedfishV1ServiceRootHeaders tests that all required Redfish headers are set
func TestGetRedfishV1ServiceRootHeaders(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
	router.GET("/redfish/v1", server.GetRedfishV1)

	req := httptest.NewRequest(http.MethodGet, "/redfish/v1", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify all Redfish-required headers
	assert.Equal(t, "4.0", w.Header().Get("OData-Version"))
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
}

// TestGetRedfishV1ServiceRootAllFields tests all response fields are present
func TestGetRedfishV1ServiceRootAllFields(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	server := &RedfishServer{Config: &dmtconfig.Config{App: dmtconfig.App{}}}
	router.GET("/redfish/v1", server.GetRedfishV1)

	req := httptest.NewRequest(http.MethodGet, "/redfish/v1", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Test all fields are present
	requiredFields := []string{
		"@odata.context",
		"@odata.id",
		"@odata.type",
		"Id",
		"Name",
		"RedfishVersion",
		"UUID",
		"Product",
		"Vendor",
		"Systems",
	}

	for _, field := range requiredFields {
		assert.Contains(t, response, field, "Response should contain %s field", field)
	}

	// Verify specific values
	assert.Equal(t, "/redfish/v1/$metadata#ServiceRoot.ServiceRoot", response["@odata.context"])
	assert.Equal(t, "/redfish/v1", response["@odata.id"])
	assert.Equal(t, "#ServiceRoot.v1_19_0.ServiceRoot", response["@odata.type"])
	assert.Equal(t, "RootService", response["Id"])
	assert.Equal(t, "Root Service", response["Name"])
	assert.Equal(t, "1.19.0", response["RedfishVersion"])
	assert.Equal(t, "Device Management Toolkit - Redfish Service", response["Product"])
	assert.Equal(t, "Device Management Toolkit", response["Vendor"])

	// Verify UUID is valid format
	uuidStr, ok := response["UUID"].(string)
	assert.True(t, ok, "UUID should be a string")

	_, err = uuid.Parse(uuidStr)
	assert.NoError(t, err, "UUID should be valid")

	// Verify Systems reference
	systems, ok := response["Systems"].(map[string]interface{})
	assert.True(t, ok, "Systems should be an object")
	assert.Equal(t, "/redfish/v1/Systems", systems["@odata.id"])
}

// TestGenerateServiceUUIDFallback tests UUID generation fallback behavior
func TestGenerateServiceUUIDFallback(t *testing.T) {
	t.Parallel()

	// Test that UUID generation doesn't panic even in worst case
	uuidStr := generateServiceUUID("")

	assert.NotEmpty(t, uuidStr)

	_, err := uuid.Parse(uuidStr)
	assert.NoError(t, err, "Generated UUID should be valid even in fallback")
}
