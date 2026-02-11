package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// Test constants
const (
	computerSystemODataType = "#ComputerSystem.v1_22_0.ComputerSystem"
	systemsEndpoint         = "/redfish/v1/Systems"
)

// Type declarations
// TestComputerSystemRepository is a test implementation of ComputerSystemRepository
type TestComputerSystemRepository struct {
	systems map[string]*redfishv1.ComputerSystem
}

func NewTestComputerSystemRepository() *TestComputerSystemRepository {
	return &TestComputerSystemRepository{
		systems: make(map[string]*redfishv1.ComputerSystem),
	}
}

func (r *TestComputerSystemRepository) AddSystem(id string, system *redfishv1.ComputerSystem) {
	r.systems[id] = system
}

func (r *TestComputerSystemRepository) GetAll(_ context.Context) ([]string, error) {
	ids := make([]string, 0, len(r.systems))
	for id := range r.systems {
		ids = append(ids, id)
	}

	return ids, nil
}

func (r *TestComputerSystemRepository) GetByID(_ context.Context, systemID string) (*redfishv1.ComputerSystem, error) {
	if system, exists := r.systems[systemID]; exists {
		return system, nil
	}

	return nil, usecase.ErrSystemNotFound
}

func (r *TestComputerSystemRepository) UpdatePowerState(_ context.Context, systemID string, resetType redfishv1.PowerState) error {
	if system, exists := r.systems[systemID]; exists {
		// Convert reset types to final power state for testing
		switch resetType {
		case redfishv1.ResetTypeOn: // Covers both ResetTypeOn and PowerStateOn (same value "On")
			system.PowerState = redfishv1.PowerStateOn
		case redfishv1.ResetTypeForceOff:
			system.PowerState = redfishv1.PowerStateOff
		case redfishv1.PowerStateOff:
			system.PowerState = redfishv1.PowerStateOff
		case redfishv1.ResetTypeForceRestart, redfishv1.ResetTypePowerCycle:
			// These reset types cycle power, for test purposes we set to Off
			// (in reality the system would restart)
			system.PowerState = redfishv1.PowerStateOff
		}

		return nil
	}

	return usecase.ErrSystemNotFound
}

func (r *TestComputerSystemRepository) GetBootSettings(_ context.Context, systemID string) (*generated.ComputerSystemBoot, error) {
	if _, exists := r.systems[systemID]; exists {
		// Return empty boot settings for tests
		return &generated.ComputerSystemBoot{}, nil
	}

	return nil, usecase.ErrSystemNotFound
}

func (r *TestComputerSystemRepository) UpdateBootSettings(_ context.Context, systemID string, _ *generated.ComputerSystemBoot) error {
	if _, exists := r.systems[systemID]; exists {
		return nil
	}

	return usecase.ErrSystemNotFound
}

// createTestSystemData creates a test system for the repository
func createTestSystemData(systemID, name, manufacturer, model, serialNumber string) *redfishv1.ComputerSystem {
	return &redfishv1.ComputerSystem{
		ID:           systemID,
		Name:         name,
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		PowerState:   redfishv1.PowerStateOn,
	}
}

// setupTestRouter sets up a test router with the given server
func setupTestRouter(server *RedfishServer) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// Register the systems endpoint
	router.GET(systemsEndpoint, server.GetRedfishV1Systems)

	return router
}

// setupTestRouterForSystemByID sets up a test router with the system by ID endpoint
func setupTestRouterForSystemByID(server *RedfishServer) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// Register the system by ID endpoint with a wrapper function
	router.GET("/redfish/v1/Systems/:ComputerSystemId", func(c *gin.Context) {
		computerSystemID := c.Param("ComputerSystemId")
		server.GetRedfishV1SystemsComputerSystemId(c, computerSystemID)
	})

	return router
}

// ====================================================================================================
// MAIN TEST FUNCTIONS
// ====================================================================================================

// TestGetRedfishV1Systems provides minimal framework test for GetRedfishV1Systems endpoint
// Detailed testing is handled in systems_test.go
func TestGetRedfishV1Systems(t *testing.T) {
	t.Parallel()

	// Setup
	testRepo := NewTestComputerSystemRepository()
	testRepo.AddSystem("System1", createTestSystemData("System1", "Test System", "Test Manufacturer", "Test Model", "SN123456"))

	useCase := &usecase.ComputerSystemUseCase{
		Repo: testRepo,
	}

	server := &RedfishServer{
		ComputerSystemUC: useCase,
	}

	router := setupTestRouter(server)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", systemsEndpoint, http.NoBody)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert basic functionality
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	// Additional assertions for concrete implementation
	assert.Contains(t, w.Body.String(), "Computer System Collection")
	assert.Contains(t, w.Body.String(), "@odata.count")
}

// TestGetRedfishV1SystemsComputerSystemId provides minimal framework test for GetRedfishV1SystemsComputerSystemId endpoint
// Detailed testing is handled in systems_test.go
func TestGetRedfishV1SystemsComputerSystemId(t *testing.T) {
	t.Parallel()

	// Setup
	testRepo := NewTestComputerSystemRepository()
	testRepo.AddSystem("550e8400-e29b-41d4-a716-446655440001", createTestSystemData("550e8400-e29b-41d4-a716-446655440001", "Test System", "Test Manufacturer", "Test Model", "SN123456"))

	useCase := &usecase.ComputerSystemUseCase{
		Repo: testRepo,
	}

	server := &RedfishServer{
		ComputerSystemUC: useCase,
	}

	router := setupTestRouterForSystemByID(server)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/redfish/v1/Systems/550e8400-e29b-41d4-a716-446655440001", http.NoBody)
	w := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(w, req)

	// Assert basic functionality
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	// Additional assertions for concrete implementation
	assert.Contains(t, w.Body.String(), "550e8400-e29b-41d4-a716-446655440001")
	assert.Contains(t, w.Body.String(), "ComputerSystem")
}
