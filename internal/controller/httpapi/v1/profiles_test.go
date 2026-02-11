package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/profiles"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// defaultValidator implements the gin binding.StructValidator interface
type defaultValidator struct {
	once     sync.Once
	validate *validator.Validate
}

func (v *defaultValidator) ValidateStruct(obj any) error {
	if obj == nil {
		return nil
	}

	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Ptr:
		if value.IsNil() {
			return nil
		}

		return v.ValidateStruct(value.Elem().Interface())
	case reflect.Struct:
		return v.validateStruct(obj)
	case reflect.Slice, reflect.Array:
		count := value.Len()
		validateRet := make(binding.SliceValidationError, 0)

		for i := 0; i < count; i++ {
			if err := v.ValidateStruct(value.Index(i).Interface()); err != nil {
				validateRet = append(validateRet, err)
			}
		}

		if len(validateRet) == 0 {
			return nil
		}

		return validateRet
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.String, reflect.UnsafePointer:
		return nil
	default:
		return nil
	}
}

func (v *defaultValidator) validateStruct(obj any) error {
	v.lazyinit()

	return v.validate.Struct(obj)
}

func (v *defaultValidator) Engine() any {
	v.lazyinit()

	return v.validate
}

func (v *defaultValidator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
		v.validate.SetTagName("binding")

		// Register custom validators
		_ = v.validate.RegisterValidation("genpasswordwone", dto.ValidateAMTPassOrGenRan)
		_ = v.validate.RegisterValidation("ciraortls", dto.ValidateCIRAOrTLS)
		_ = v.validate.RegisterValidation("wifidhcp", dto.ValidateWiFiDHCP)
	})
}

func profilesTest(t *testing.T) (*mocks.MockProfilesFeature, *gin.Engine) {
	t.Helper()

	// Enable validation for tests
	if binding.Validator == nil {
		binding.Validator = &defaultValidator{}
	}

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	log := logger.New("error")
	mockProfiles := mocks.NewMockProfilesFeature(mockCtl)

	engine := gin.New()
	handler := engine.Group("/api/v1/admin")

	NewProfileRoutes(handler, mockProfiles, log)

	return mockProfiles, engine
}

type testProfiles struct {
	name         string
	method       string
	url          string
	mock         func(repo *mocks.MockProfilesFeature)
	response     interface{}
	requestBody  dto.Profile
	expectedCode int
}

var profileTest = dto.Profile{
	ProfileName:                "newprofile",
	AMTPassword:                "P@ssw0rd",
	GenerateRandomPassword:     false,
	CIRAConfigName:             nil,
	Activation:                 "ccmactivate",
	MEBXPassword:               "",
	GenerateRandomMEBxPassword: false,
	CIRAConfigObject:           nil,
	Tags:                       nil,
	DHCPEnabled:                false,
	IPSyncEnabled:              false,
	LocalWiFiSyncEnabled:       false,
	WiFiConfigs:                nil,
	TenantID:                   "",
	TLSMode:                    0,
	TLSCerts:                   nil,
	TLSSigningAuthority:        "",
	UserConsent:                "",
	IDEREnabled:                false,
	KVMEnabled:                 false,
	SOLEnabled:                 false,
	IEEE8021xProfileName:       nil,
	IEEE8021xProfile:           nil,
	Version:                    "1.0",
	UEFIWiFiSyncEnabled:        false,
}

func TestProfileRoutes(t *testing.T) { //nolint:gocognit,paralleltest // this is a test function
	tests := []testProfiles{
		{
			name:   "get all profiles",
			method: http.MethodGet,
			url:    "/api/v1/admin/profiles",
			mock: func(profile *mocks.MockProfilesFeature) {
				profile.EXPECT().Get(context.Background(), 25, 0, "").Return([]dto.Profile{{
					ProfileName: "profile",
				}}, nil)
			},
			response:     []dto.Profile{{ProfileName: "profile"}},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get all profiles - with count",
			method: http.MethodGet,
			url:    "/api/v1/admin/profiles?$top=10&$skip=1&$count=true",
			mock: func(domain *mocks.MockProfilesFeature) {
				domain.EXPECT().Get(context.Background(), 10, 1, "").Return([]dto.Profile{{
					ProfileName: "profile",
				}}, nil)
				domain.EXPECT().GetCount(context.Background(), "").Return(1, nil)
			},
			response:     dto.ProfileCountResponse{Count: 1, Data: []dto.Profile{{ProfileName: "profile"}}},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get all profiles - failed",
			method: http.MethodGet,
			url:    "/api/v1/admin/profiles",
			mock: func(domain *mocks.MockProfilesFeature) {
				domain.EXPECT().Get(context.Background(), 25, 0, "").Return(nil, profiles.ErrDatabase)
			},
			response:     profiles.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "get profile by name",
			method: http.MethodGet,
			url:    "/api/v1/admin/profiles/profile",
			mock: func(profile *mocks.MockProfilesFeature) {
				profile.EXPECT().GetByName(context.Background(), "profile", "").Return(&dto.Profile{
					ProfileName: "profile",
				}, nil)
			},
			response:     dto.Profile{ProfileName: "profile"},
			expectedCode: http.StatusOK,
		},
		{
			name:   "get profile by name - failed",
			method: http.MethodGet,
			url:    "/api/v1/admin/profiles/profile",
			mock: func(profile *mocks.MockProfilesFeature) {
				profile.EXPECT().GetByName(context.Background(), "profile", "").Return(nil, profiles.ErrDatabase)
			},
			response:     profiles.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "insert profile",
			method: http.MethodPost,
			url:    "/api/v1/admin/profiles",
			mock: func(profile *mocks.MockProfilesFeature) {
				profile.EXPECT().Insert(context.Background(), &profileTest).Return(&profileTest, nil)
			},
			response:     profileTest,
			requestBody:  profileTest,
			expectedCode: http.StatusCreated,
		},
		{
			name:   "insert profile - failed",
			method: http.MethodPost,
			url:    "/api/v1/admin/profiles",
			mock: func(profile *mocks.MockProfilesFeature) {
				profile.EXPECT().Insert(context.Background(), &profileTest).Return(nil, profiles.ErrDatabase)
			},
			response:     profiles.ErrDatabase,
			requestBody:  profileTest,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "delete profile",
			method: http.MethodDelete,
			url:    "/api/v1/admin/profiles/profile",
			mock: func(profile *mocks.MockProfilesFeature) {
				profile.EXPECT().Delete(context.Background(), "profile", "").Return(nil)
			},
			response:     nil,
			expectedCode: http.StatusNoContent,
		},
		{
			name:   "delete profile - failed",
			method: http.MethodDelete,
			url:    "/api/v1/admin/profiles/profile",
			mock: func(profile *mocks.MockProfilesFeature) {
				profile.EXPECT().Delete(context.Background(), "profile", "").Return(profiles.ErrDatabase)
			},
			response:     profiles.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "update profile",
			method: http.MethodPatch,
			url:    "/api/v1/admin/profiles",
			mock: func(profile *mocks.MockProfilesFeature) {
				profile.EXPECT().Update(context.Background(), &profileTest).Return(&profileTest, nil)
			},
			response:     profileTest,
			requestBody:  profileTest,
			expectedCode: http.StatusOK,
		},
		{
			name:   "update profile - failed",
			method: http.MethodPatch,
			url:    "/api/v1/admin/profiles",
			mock: func(profile *mocks.MockProfilesFeature) {
				profile.EXPECT().Update(context.Background(), &profileTest).Return(nil, profiles.ErrDatabase)
			},
			response:     profiles.ErrDatabase,
			requestBody:  profileTest,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "export profile successfully",
			method: http.MethodGet,
			url:    "/api/v1/admin/profiles/export/profile?domainName=test.com",
			mock: func(profile *mocks.MockProfilesFeature) {
				profile.EXPECT().Export(context.Background(), "profile", "test.com", "").Return(
					"yaml-content",   // content
					"encryption-key", // key
					nil,              // error
				)
			},
			response: gin.H{
				"filename": "profile.yaml",
				"content":  "yaml-content",
				"key":      "encryption-key",
			},
			expectedCode: http.StatusOK,
		},
		{
			name:   "export profile - failed",
			method: http.MethodGet,
			url:    "/api/v1/admin/profiles/export/profile?domainName=test.com",
			mock: func(profile *mocks.MockProfilesFeature) {
				profile.EXPECT().Export(
					context.Background(),
					"profile",
					"test.com",
					"",
				).Return(
					"", // empty content
					"", // empty key
					profiles.ErrDatabase,
				)
			},
			response:     profiles.ErrDatabase,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:   "export profile with no domain",
			method: http.MethodGet,
			url:    "/api/v1/admin/profiles/export/profile",
			mock: func(profile *mocks.MockProfilesFeature) {
				profile.EXPECT().Export(
					context.Background(),
					"profile",
					"",
					"",
				).Return(
					"yaml-content",   // content
					"encryption-key", // key
					nil,              // error
				)
			},
			response: gin.H{
				"filename": "profile.yaml",
				"content":  "yaml-content",
				"key":      "encryption-key",
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range tests { //nolint:paralleltest // tests run sequentially for simplicity
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			profileFeature, engine := profilesTest(t)

			tc.mock(profileFeature)

			var req *http.Request

			var err error

			if tc.requestBody.ProfileName != "" {
				reqBody, _ := json.Marshal(tc.requestBody)
				req, err = http.NewRequestWithContext(context.Background(), tc.method, tc.url, bytes.NewBuffer(reqBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequestWithContext(context.Background(), tc.method, tc.url, http.NoBody)
			}

			if err != nil {
				t.Fatalf("Couldn't create request: %v\n", err)
			}

			w := httptest.NewRecorder()

			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code)

			if tc.expectedCode == http.StatusOK || tc.expectedCode == http.StatusCreated {
				if response, ok := tc.response.(gin.H); ok {
					// For gin.H responses (like from export)
					var actualResponse gin.H

					err := json.Unmarshal(w.Body.Bytes(), &actualResponse)
					require.NoError(t, err)
					require.Equal(t, response, actualResponse)
				} else {
					// For other responses
					jsonBytes, err := json.Marshal(tc.response)
					require.NoError(t, err)
					require.Equal(t, string(jsonBytes), w.Body.String())
				}
			}
		})
	}
}

func TestProfileValidation(t *testing.T) { //nolint:paralleltest // tests run sequentially for simplicity
	tests := []struct {
		name         string
		profile      dto.Profile
		expectedCode int
	}{
		{
			name: "valid profile - CCM with CIRA",
			profile: dto.Profile{
				ProfileName:                "test-profile",
				Activation:                 "ccmactivate",
				GenerateRandomPassword:     true,
				GenerateRandomMEBxPassword: true,
				CIRAConfigName:             stringPtr("cira-config"),
				DHCPEnabled:                true,
				UserConsent:                "All",
				TenantID:                   "tenant1",
			},
			expectedCode: http.StatusCreated,
		},
		{
			name: "valid profile - ACM with TLS",
			profile: dto.Profile{
				ProfileName:                "test-profile",
				Activation:                 "acmactivate",
				GenerateRandomPassword:     true,
				MEBXPassword:               "P@ssw0rd123",
				GenerateRandomMEBxPassword: false,
				TLSMode:                    1,
				TLSSigningAuthority:        "SelfSigned",
				DHCPEnabled:                true,
				UserConsent:                "KVM",
				TenantID:                   "tenant1",
			},
			expectedCode: http.StatusCreated,
		},
		{
			name: "invalid - both CIRA and TLS",
			profile: dto.Profile{
				ProfileName:                "test-profile",
				Activation:                 "ccmactivate",
				GenerateRandomPassword:     true,
				GenerateRandomMEBxPassword: true,
				CIRAConfigName:             stringPtr("cira-config"),
				TLSMode:                    1,
				DHCPEnabled:                true,
				UserConsent:                "All",
				TenantID:                   "tenant1",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid - wifi configs without DHCP",
			profile: dto.Profile{
				ProfileName:                "test-profile",
				Activation:                 "ccmactivate",
				GenerateRandomPassword:     true,
				GenerateRandomMEBxPassword: true,
				DHCPEnabled:                false,
				WiFiConfigs: []dto.ProfileWiFiConfigs{
					{ProfileName: "wifi1", Priority: 1},
				},
				UserConsent: "All",
				TenantID:    "tenant1",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid - password set with genRandom true",
			profile: dto.Profile{
				ProfileName:                "test-profile",
				Activation:                 "ccmactivate",
				AMTPassword:                "P@ssw0rd123",
				GenerateRandomPassword:     true,
				GenerateRandomMEBxPassword: true,
				DHCPEnabled:                true,
				UserConsent:                "All",
				TenantID:                   "tenant1",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid - invalid activation",
			profile: dto.Profile{
				ProfileName:                "test-profile",
				Activation:                 "invalidactivation",
				GenerateRandomPassword:     true,
				GenerateRandomMEBxPassword: true,
				UserConsent:                "All",
				TenantID:                   "tenant1",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid - invalid TLS signing authority",
			profile: dto.Profile{
				ProfileName:                "test-profile",
				Activation:                 "acmactivate",
				GenerateRandomPassword:     true,
				GenerateRandomMEBxPassword: true,
				TLSMode:                    1,
				TLSSigningAuthority:        "InvalidAuthority",
				DHCPEnabled:                true,
				UserConsent:                "All",
				TenantID:                   "tenant1",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid - TLS mode out of range",
			profile: dto.Profile{
				ProfileName:                "test-profile",
				Activation:                 "acmactivate",
				GenerateRandomPassword:     true,
				GenerateRandomMEBxPassword: true,
				TLSMode:                    5,
				TLSSigningAuthority:        "SelfSigned",
				DHCPEnabled:                true,
				UserConsent:                "All",
				TenantID:                   "tenant1",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid - password too short",
			profile: dto.Profile{
				ProfileName:                "test-profile",
				Activation:                 "acmactivate",
				AMTPassword:                "short",
				GenerateRandomPassword:     false,
				MEBXPassword:               "P@ssw0rd123",
				GenerateRandomMEBxPassword: false,
				DHCPEnabled:                true,
				UserConsent:                "All",
				TenantID:                   "tenant1",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid - password missing special character",
			profile: dto.Profile{
				ProfileName:                "test-profile",
				Activation:                 "acmactivate",
				AMTPassword:                "Password123",
				GenerateRandomPassword:     false,
				MEBXPassword:               "P@ssw0rd123",
				GenerateRandomMEBxPassword: false,
				DHCPEnabled:                true,
				UserConsent:                "All",
				TenantID:                   "tenant1",
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests { //nolint:paralleltest // tests run sequentially for simplicity
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			profileFeature, engine := profilesTest(t)

			if tc.expectedCode == http.StatusCreated {
				profileFeature.EXPECT().Insert(context.Background(), &tc.profile).Return(&tc.profile, nil)
			}

			reqBody, _ := json.Marshal(tc.profile)
			req, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				"/api/v1/admin/profiles",
				bytes.NewBuffer(reqBody),
			)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			require.Equal(t, tc.expectedCode, w.Code)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
