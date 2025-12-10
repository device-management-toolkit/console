package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/profiles"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func profilesTest(t *testing.T) (*mocks.MockProfilesFeature, *gin.Engine) {
	t.Helper()

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

func TestProfileRoutes(t *testing.T) { //nolint:gocognit // this is a test function
	t.Parallel()

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

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			profileFeature, engine := profilesTest(t)

			tc.mock(profileFeature)

			var req *http.Request

			var err error

			if tc.requestBody.ProfileName != "" {
				reqBody, _ := json.Marshal(tc.requestBody)
				req, err = http.NewRequestWithContext(context.Background(), tc.method, tc.url, bytes.NewBuffer(reqBody))
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
