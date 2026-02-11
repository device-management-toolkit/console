// Package v1 provides Redfish v1 API route setup and configuration.
package v1

import (
	"embed"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"gopkg.in/yaml.v2"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
)

// ServiceRoot OData metadata constants
const (
	odataContextServiceRoot = "/redfish/v1/$metadata#ServiceRoot.ServiceRoot"
	odataIDServiceRoot      = "/redfish/v1"
	odataTypeServiceRoot    = "#ServiceRoot.v1_19_0.ServiceRoot"
	serviceRootID           = "RootService"
	serviceRootName         = "Root Service"
	redfishVersion          = "1.19.0"
)

// ODataService represents a Redfish service in the OData service document
type ODataService struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

// Metadata and OpenAPI spec embedding and caching
//
//go:embed metadata.xml
var metadataFS embed.FS

var (
	metadataXML    string
	metadataLoaded bool
	metadataMutex  sync.Mutex
	cachedUUID     string
	uuidMutex      sync.Mutex
)

// loadMetadata loads embedded metadata.xml with XML validation.
// This function is thread-safe and ensures metadata is loaded only once.
// The metadata file is embedded at build time using go:embed directive
// from the current package.
func loadMetadata() {
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	// Double-check locking pattern
	if metadataLoaded {
		return
	}

	data, err := metadataFS.ReadFile("metadata.xml")
	if err != nil {
		log.Warnf("Could not load embedded metadata.xml: %v", err)

		return
	}

	metadataXML = string(data)

	// Validate XML
	if err := validateMetadataXML(metadataXML); err != nil {
		log.Warnf("Invalid metadata.xml: %v", err)

		return
	}

	log.Infof("Embedded metadata.xml loaded and validation passed")

	metadataLoaded = true
}

// validateMetadataXML checks if the provided XML string is well-formed.
func validateMetadataXML(xmlData string) error {
	if xmlData == "" {
		return nil // Empty is acceptable
	}

	var doc interface{}

	if err := xml.Unmarshal([]byte(xmlData), &doc); err != nil {
		return fmt.Errorf("XML parsing failed: %w", err)
	}

	return nil
}

const uuidFileName = "service_uuid"

// getUUIDStoragePath returns the OS-agnostic path for storing the service UUID.
// Works across Linux, Windows, and macOS using the user config directory.
func getUUIDStoragePath(appName string) (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	dir := filepath.Join(base, appName)

	const dirPermissions = 0o755

	if err := os.MkdirAll(dir, dirPermissions); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(dir, uuidFileName), nil
}

// loadOrCreateUUID loads an existing UUID from file or creates a new one.
// This ensures the UUID persists across service restarts.
func loadOrCreateUUID(appName string) (string, error) {
	file, err := getUUIDStoragePath(appName)
	if err != nil {
		return "", err
	}

	// Try to read existing UUID
	if data, err := os.ReadFile(file); err == nil {
		storedUUID := string(data)
		// Validate it's a proper UUID
		if _, err := uuid.Parse(storedUUID); err == nil {
			return storedUUID, nil
		}

		log.Warnf("Invalid UUID in storage file, generating new one")
	}

	// Create new UUID
	newUUID := uuid.New().String()

	const filePermissions = 0o600

	// Save to file
	if err := os.WriteFile(file, []byte(newUUID), filePermissions); err != nil {
		log.Warnf("Failed to save UUID to file: %v", err)
		// Continue with the generated UUID even if save fails
	}

	return newUUID, nil
}

// generateServiceUUID generates or retrieves the service instance UUID.
// Per Redfish specification, this UUID should be consistent across service restarts.
// Priority order:
// 1. Environment UUID from configuration (REDFISH_ENV_UUID)
// 2. Cached UUID in memory (for process lifetime)
// 3. Persisted UUID from config file
// 4. Newly generated UUID (saved to config file for future use)
func generateServiceUUID(envUUID string) string {
	uuidMutex.Lock()
	defer uuidMutex.Unlock()

	// Priority 1: Use environment UUID if configured
	if envUUID != "" {
		// Validate it's a proper UUID
		if _, err := uuid.Parse(envUUID); err == nil {
			cachedUUID = envUUID

			return envUUID
		}

		log.Warnf("Invalid environment UUID configured: %s, falling back to persistent UUID", envUUID)
	}

	// Priority 2: Return cached UUID if available
	if cachedUUID != "" {
		return cachedUUID
	}

	// Priority 3: Load or create persistent UUID
	serviceUUID, err := loadOrCreateUUID("dmt-redfish-service")
	if err != nil {
		log.Warnf("Failed to load/create persistent UUID: %v, generating temporary UUID", err)
		// Fallback to temporary UUID for this session (but cache it)
		cachedUUID = uuid.New().String()

		return cachedUUID
	}

	// Cache the UUID for this process lifetime
	cachedUUID = serviceUUID

	return serviceUUID
}

// GetRedfishV1 returns the service root
// ExtractServicesFromOpenAPIData parses the embedded OpenAPI spec data and extracts all top-level Redfish services
// Automatically discovers services from /redfish/v1/* paths in the spec
func ExtractServicesFromOpenAPIData(data []byte) ([]ODataService, error) {
	var spec map[string]interface{}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		log.Warnf("Could not parse OpenAPI spec: %v", err)

		return GetDefaultServices(), nil
	}

	pathsObj, ok := spec["paths"].(map[interface{}]interface{})
	if !ok {
		log.Warnf("No paths found in OpenAPI spec")

		return GetDefaultServices(), nil
	}

	serviceMap := make(map[string]ODataService)

	for path := range pathsObj {
		pathStr, ok := path.(string)
		if !ok {
			continue
		}

		// Extract top-level services only: /redfish/v1/Systems, /redfish/v1/SessionService, etc.
		// Skip parametrized paths like /redfish/v1/Systems/{ComputerSystemId}
		// Skip sub-resources like /redfish/v1/SessionService/Sessions
		if strings.HasPrefix(pathStr, "/redfish/v1/") && !strings.Contains(pathStr, "{") {
			// Extract service name (e.g., "Systems" from "/redfish/v1/Systems")
			serviceName := strings.TrimPrefix(pathStr, "/redfish/v1/")

			// Skip metadata, odata endpoints, root path (empty name), and sub-resources (containing /)
			if serviceName != "" && serviceName != "odata" && serviceName != "$metadata" && !strings.Contains(serviceName, "/") {
				serviceMap[serviceName] = ODataService{
					Name: serviceName,
					Kind: "Singleton",
					URL:  pathStr,
				}
			}
		}
	}

	// Convert map to sorted slice for consistent ordering
	var services []ODataService

	var names []string

	for name := range serviceMap {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		services = append(services, serviceMap[name])
	}

	if len(services) == 0 {
		log.Warnf("No services found in OpenAPI spec, using defaults")

		return GetDefaultServices(), nil
	}

	log.Infof("Loaded %d services from OpenAPI spec: %v", len(services), names)

	return services, nil
}

// GetDefaultServices returns the standard Redfish services as fallback
func GetDefaultServices() []ODataService {
	return []ODataService{
		{Name: "Systems", Kind: "Singleton", URL: "/redfish/v1/Systems"},
	}
}

// Path: GET /redfish/v1
// Spec: Redfish ServiceRoot.v1_19_0
// This is the entry point for the Redfish API, providing links to all available resources.
// Per Redfish specification, this endpoint must be accessible without authentication.
func (s *RedfishServer) GetRedfishV1(c *gin.Context) {
	// Set Redfish-compliant headers
	SetRedfishHeaders(c)

	type ServiceRootWithSessionService struct {
		generated.ServiceRootServiceRoot
		SessionService *generated.OdataV4IdRef `json:"SessionService,omitempty"`
	}

	// Create Links with Sessions for redfishtool compatibility
	links := generated.ServiceRootLinks{
		"Sessions": map[string]interface{}{
			"@odata.id": "/redfish/v1/SessionService/Sessions",
		},
	}

	serviceRoot := ServiceRootWithSessionService{
		ServiceRootServiceRoot: generated.ServiceRootServiceRoot{
			OdataContext:   StringPtr(odataContextServiceRoot),
			OdataId:        StringPtr(odataIDServiceRoot),
			OdataType:      StringPtr(odataTypeServiceRoot),
			Id:             serviceRootID,
			Name:           serviceRootName,
			RedfishVersion: StringPtr(redfishVersion),
			UUID:           StringPtr(generateServiceUUID("")),
			Product:        StringPtr("Device Management Toolkit - Redfish Service"),
			Vendor:         StringPtr("Device Management Toolkit"),
			Links:          &links,
			Systems: &generated.OdataV4IdRef{
				OdataId: StringPtr("/redfish/v1/Systems"),
			},
		},
		SessionService: &generated.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/SessionService"),
		},
	}

	c.JSON(http.StatusOK, serviceRoot)
}

// GetRedfishV1Metadata returns the OData CSDL metadata document describing the service's data model.
// Path: GET /redfish/v1/$metadata
// Spec: OData CSDL v4.0 - Redfish specification mandates this endpoint is accessible without authentication.
func (s *RedfishServer) GetRedfishV1Metadata(c *gin.Context) {
	// Ensure metadata is loaded
	loadMetadata()

	// Read metadata safely
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	metadata := metadataXML

	// Set Redfish-compliant headers
	c.Header(headerContentType, contentTypeXML)
	c.Header(headerODataVersion, odataVersion)
	c.String(http.StatusOK, metadata)
}

// GetRedfishV1Odata returns the OData service document listing available Redfish services
// Path: GET /redfish/v1/odata
// Spec: OData v4.0 Service Document - DMTF Redfish compliant (services auto-discovered from OpenAPI spec)
func (s *RedfishServer) GetRedfishV1Odata(c *gin.Context) {
	SetRedfishHeaders(c)

	// Use cached services or fallback to defaults if not loaded
	services := s.Services
	if len(services) == 0 {
		services = GetDefaultServices()
	}

	response := map[string]interface{}{
		"@odata.context": odataContextServiceRoot,
		"value":          services,
	}
	c.JSON(http.StatusOK, response)
}
