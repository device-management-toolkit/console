package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

func FuzzGetRedfishV1SystemsComputerSystemIdHandler(f *testing.F) {
	seedInputs := []string{
		testUUID1,
		"",
		testUUIDNotFound,
		"not-a-uuid",
		"../etc/passwd",
		"用戶🙂",
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	f.Fuzz(func(t *testing.T, systemID string) {
		repo := NewTestSystemsComputerSystemRepository()
		repo.AddSystem(testUUID1, &redfishv1.ComputerSystem{
			ID:           testUUID1,
			Name:         "Test System",
			PowerState:   redfishv1.PowerStateOn,
			Manufacturer: "TestMfg",
			Model:        "TestModel",
			SerialNumber: "SN12345",
			SystemType:   redfishv1.SystemTypePhysical,
		})

		server := &RedfishServer{ComputerSystemUC: &usecase.ComputerSystemUseCase{Repo: repo}}
		router := setupSystemByIDTestRouter(server)

		req := httptest.NewRequest(http.MethodGet, "/redfish/v1/Systems/"+url.PathEscape(systemID), http.NoBody)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		switch w.Code {
		case http.StatusOK, http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError:
		default:
			t.Fatalf("unexpected status %d for systemID %q", w.Code, systemID)
		}

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse successful system response: %v", err)
			}
		}
	})
}

func FuzzResetActionHandler(f *testing.F) {
	seedInputs := []struct {
		systemID string
		body     string
	}{
		{testSystemID, `{"ResetType":"On"}`},
		{testSystemID, `{"ResetType":"ForceOff"}`},
		{testSystemID, `{"ResetType":"invalid"}`},
		{testUUIDNotFound, `{"ResetType":"On"}`},
		{"not-a-uuid", `{"ResetType":"On"}`},
		{testSystemID, `{}`},
		{testSystemID, `not-json`},
	}

	for _, input := range seedInputs {
		f.Add(input.systemID, input.body)
	}

	f.Fuzz(func(t *testing.T, systemID, body string) {
		repo := NewTestSystemsComputerSystemRepository()
		repo.AddSystem(testSystemID, &redfishv1.ComputerSystem{
			ID:           testSystemID,
			Name:         "Test System",
			PowerState:   redfishv1.PowerStateOn,
			Manufacturer: "TestMfg",
			Model:        "TestModel",
			SerialNumber: "SN12345",
		})

		server := setupSystemActionsTestServer(repo)
		router := setupSystemActionsTestRouter(server)

		req := httptest.NewRequest(http.MethodPost, "/redfish/v1/Systems/"+url.PathEscape(systemID)+"/Actions/ComputerSystem.Reset", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		switch w.Code {
		case http.StatusAccepted, http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusInternalServerError:
		default:
			t.Fatalf("unexpected status %d for systemID %q body %q", w.Code, systemID, body)
		}

		if w.Code == http.StatusAccepted {
			var response map[string]interface{}

			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse accepted task response: %v", err)
			}

			if response["TaskState"] != taskStateCompleted {
				t.Fatalf("expected task state %q, got %v", taskStateCompleted, response["TaskState"])
			}
		}
	})
}

func FuzzResetActionRequestBody(f *testing.F) {
	seedInputs := []string{
		string(generated.ResourceResetTypeOn),
		string(generated.ResourceResetTypeForceOff),
		string(generated.ResourceResetTypeForceRestart),
		"",
		"invalid",
		"用戶🙂",
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	f.Fuzz(func(t *testing.T, resetType string) {
		payload, err := json.Marshal(map[string]string{"ResetType": resetType})
		if err != nil {
			t.Fatalf("failed to marshal reset type payload: %v", err)
		}

		var request generated.PostRedfishV1SystemsComputerSystemIdActionsComputerSystemResetJSONRequestBody

		firstErr := json.Unmarshal(payload, &request)
		secondErr := json.Unmarshal(payload, &request)

		if (firstErr == nil) != (secondErr == nil) {
			t.Fatalf("non-deterministic request parse for resetType %q", resetType)
		}
	})
}
