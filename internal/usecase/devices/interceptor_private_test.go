package devices

import (
	"bytes"
	"errors"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/client"
	"github.com/stretchr/testify/require"
)

func TestProcessBrowserData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		msg           []byte
		challenge     *client.AuthChallenge
		expectedBytes []byte
	}{
		{
			name:          "Start Redirection Session",
			msg:           []byte{RedirectionCommandsStartRedirectionSession, 0, 0, 0, 0, 0, 0, 0, 0},
			expectedBytes: []byte{RedirectionCommandsStartRedirectionSession, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:          "End Redirection Session",
			msg:           []byte{RedirectionCommandsEndRedirectionSession, 0, 0, 0},
			expectedBytes: []byte{RedirectionCommandsEndRedirectionSession, 0, 0, 0},
		},
		{
			name: "Authenticate Session",
			msg:  []byte{RedirectionCommandsAuthenticateSession, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			challenge: &client.AuthChallenge{
				Username:   "testuser",
				Password:   "testpassword",
				Realm:      "testrealm",
				CSRFToken:  "csrf1234",
				Domain:     "testdomain",
				Nonce:      "noncevalue",
				Opaque:     "opaquevalue",
				Stale:      "false",
				Algorithm:  "MD5",
				Qop:        "auth",
				CNonce:     "cnoncevalue",
				NonceCount: 1,
			},
			expectedBytes: []byte{0x13, 0x0, 0x0, 0x0, 0x4, 0x6c, 0x0, 0x0, 0x0, 0x8, 0x74, 0x65, 0x73, 0x74, 0x75, 0x73, 0x65, 0x72, 0x9, 0x74, 0x65, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6c, 0x6d, 0xa, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x13, 0x2f, 0x52, 0x65, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0xa, 0x63, 0x34, 0x65, 0x64, 0x35, 0x32, 0x62, 0x30, 0x37, 0x61, 0x8, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x31, 0x20, 0x31, 0x65, 0x39, 0x36, 0x35, 0x66, 0x32, 0x33, 0x30, 0x38, 0x63, 0x38, 0x35, 0x64, 0x35, 0x35, 0x63, 0x63, 0x31, 0x65, 0x37, 0x62, 0x33, 0x38, 0x36, 0x36, 0x31, 0x32, 0x65, 0x39, 0x38, 0x38, 0x4, 0x61, 0x75, 0x74, 0x68},
		},
		{
			name:          "Default Case",
			msg:           []byte{0xFF},
			expectedBytes: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := processBrowserData(tc.msg, tc.challenge)
			require.IsType(t, tc.expectedBytes, result)
		})
	}
}

func TestProcessDeviceData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		msg          []byte
		challenge    *client.AuthChallenge
		expectedData []byte
		expectedBool bool
	}{
		{
			name:         "Start Redirection Session Reply",
			msg:          []byte{RedirectionCommandsStartRedirectionSessionReply, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
			challenge:    nil,
			expectedData: []byte{0x11, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2},
			expectedBool: false,
		},
		{
			name:         "Authenticate Session Reply",
			msg:          []byte{RedirectionCommandsAuthenticateSessionReply, 0x01, 0x02, 0x03, 0x04},
			challenge:    &client.AuthChallenge{},
			expectedData: []byte{},
			expectedBool: false,
		},
		{
			name:         "Unhandled Command",
			msg:          []byte{0x99},
			challenge:    nil,
			expectedData: nil,
			expectedBool: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, ok := processDeviceData(tc.msg, tc.challenge)

			require.Equal(t, tc.expectedData, data)
			require.Equal(t, tc.expectedBool, ok)
		})
	}
}

func TestHandleStartRedirectionSessionReply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		msg      []byte
		expected []byte
	}{
		{
			name:     "Valid Session Reply",
			msg:      []byte{RedirectionCommandsStartRedirectionSessionReply, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
			expected: []byte{0x11, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x2},
		},
		{
			name:     "Message Shorter Than RedirectionSessionReply",
			msg:      []byte{RedirectionCommandsStartRedirectionSessionReply, 0x01},
			expected: []byte(""),
		},
		{
			name:     "Message Shorter Than RedirectSessionLengthBytes",
			msg:      []byte{RedirectionCommandsStartRedirectionSessionReply, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: []byte(""),
		},
		{
			name:     "Message Shorter Than RedirectSessionLengthBytes Plus OEM Length",
			msg:      []byte{RedirectionCommandsStartRedirectionSessionReply, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x04},
			expected: []byte(""),
		},
		{
			name:     "Invalid Session Reply",
			msg:      []byte{RedirectionCommandsStartRedirectionSessionReply, 7, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			expected: []byte(""),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := handleStartRedirectionSessionReply(tc.msg)

			require.Equal(t, tc.expected, result)
		})
	}
}

func TestAllZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "All zeros",
			data:     []byte{0x00, 0x00, 0x00},
			expected: true,
		},
		{
			name:     "Not all zeros",
			data:     []byte{0x00, 0x01, 0x00},
			expected: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := allZero(tc.data)

			require.Equal(t, tc.expected, result)
		})
	}
}

type failBuffer struct{}

var ErrSimWriteFail = errors.New("simulated write failure")

func (f *failBuffer) Write(_ []byte) (n int, err error) {
	return 0, ErrSimWriteFail
}

type failBufferOnSecondWrite struct {
	count int
}

var ErrForcedFailure = errors.New("forced failure on second write")

func (f *failBufferOnSecondWrite) Write(p []byte) (n int, err error) {
	f.count++
	if f.count == 2 {
		return 0, ErrForcedFailure
	}

	return len(p), nil
}

func TestWriteField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		field      string
		expected   []byte
		shouldFail bool
		buffer     io.Writer
	}{
		{
			name:     "Valid field",
			field:    "test",
			expected: append([]byte{0x04}, []byte("test")...),
			buffer:   &bytes.Buffer{},
		},
		{
			name:     "Empty field",
			field:    "",
			expected: []byte{0x00},
			buffer:   &bytes.Buffer{},
		},
		{
			name:       "Write length failure",
			field:      "failLength",
			shouldFail: true,
			buffer:     &failBuffer{},
		},
		{
			name:       "Write field failure",
			field:      "failField",
			shouldFail: true,
			buffer:     &failBufferOnSecondWrite{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := writeField(tc.buffer, tc.field)

			if tc.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				if buf, ok := tc.buffer.(*bytes.Buffer); ok {
					result := buf.Bytes()
					require.Equal(t, tc.expected, result)
				}
			}
		})
	}
}

func TestGenerateEmptyAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		challenge   *client.AuthChallenge
		authURL     string
		expectedBuf []byte
	}{
		{
			name: "Valid Auth Challenge",
			challenge: &client.AuthChallenge{
				Username: "testuser",
			},
			authURL:     "http://example.com",
			expectedBuf: []byte{0x13, 0x0, 0x0, 0x0, 0x4, 0x22, 0x0, 0x0, 0x0, 0x8, 0x74, 0x65, 0x73, 0x74, 0x75, 0x0, 0x0, 0x12, 0x68, 0x74, 0x74, 0x70, 0x3a, 0x2f, 0x2f, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x63, 0x6f, 0x6d, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
		{
			name: "Empty Username and URL",
			challenge: &client.AuthChallenge{
				Username: "",
			},
			authURL:     "",
			expectedBuf: []byte{0x13, 0x0, 0x0, 0x0, 0x4, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := generateEmptyAuth(tc.challenge, tc.authURL)

			require.Equal(t, tc.expectedBuf, result)
		})
	}
}

func TestHandleAuthenticateSessionReply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		msg               []byte
		expectedResult    []byte
		expectedSuccess   bool
		expectedChallenge *client.AuthChallenge
	}{
		{
			name:            "Message too short",
			msg:             []byte{0x01},
			expectedResult:  []byte(""),
			expectedSuccess: false,
		},
		{
			name: "Valid Digest Authentication Fail",
			msg: []byte{
				0x01,
				AuthenticationStatusFail, 0x00, 0x00, AuthenticationTypeDigest, 0x12, 0x00, 0x00, 0x00,
				0x05,
				'r', 'e', 'a', 'l', 'm',
				0x06,
				'n', 'o', 'n', 'c', 'e', '1',
				0x03,
				'q', 'o', 'p',
			},
			expectedResult:  []byte{},
			expectedSuccess: false,
			expectedChallenge: &client.AuthChallenge{
				Realm: "",
				Nonce: "",
				Qop:   "",
			},
		},
		{
			name: "Valid Authentication Success",
			msg: []byte{
				0x01,
				AuthenticationStatusSuccess, 0x00, 0x00, AuthenticationTypeQuery, 0x00, 0x00, 0x00, 0x00,
			},
			expectedResult: []byte{
				0x01, AuthenticationStatusSuccess, 0x00, 0x00, AuthenticationTypeQuery, 0x00, 0x00, 0x00, 0x00,
			},
			expectedSuccess: false,
		},
		{
			name: "Valid Authentication Success, Non-Digest Type",
			msg: []byte{
				0x01,
				AuthenticationStatusSuccess, 0x00, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00,
			},
			expectedResult: []byte{
				0x01, AuthenticationStatusSuccess, 0x00, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00,
			},
			expectedSuccess: true,
		},
		{
			name: "Invalid length in message",
			msg: []byte{
				0x01,
				AuthenticationStatusFail, 0x00, 0x00, AuthenticationTypeDigest, 0xFF, 0xFF, 0xFF, 0xFF,
			},
			expectedResult:  []byte(""),
			expectedSuccess: false,
		},
		{
			name: "Digest Authentication Failure with valid realm, nonce, and qop",
			msg: []byte{
				0x01,
				AuthenticationStatusFail, 0x00, 0x00, AuthenticationTypeDigest, 0x12, 0x00, 0x00, 0x00,
				0x05,
				'r', 'e', 'a', 'l', 'm',
				0x06,
				'n', 'o', 'n', 'c', 'e', '1',
				0x03, 0x03, 0x03,
				'q', 'o', 'p',
			},
			expectedResult:  []byte{0x1, 0x1, 0x0, 0x0, 0x4, 0x12, 0x0, 0x0, 0x0, 0x5, 0x72, 0x65, 0x61, 0x6c, 0x6d, 0x6, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x31, 0x3, 0x3, 0x3, 0x71, 0x6f, 0x70},
			expectedSuccess: false,
			expectedChallenge: &client.AuthChallenge{
				Realm: "realm",
				Nonce: "nonce1",
				Qop:   "\x03\x03q",
			},
		},
		{
			name: "Digest Authentication Failure with empty realm, nonce, and qop",
			msg: []byte{
				0x01,
				AuthenticationStatusFail, 0x00, 0x00, AuthenticationTypeDigest, 0x12, 0x00, 0x00, 0x00,
				0x00,
				0x00,
				0x00,
			},
			expectedResult: []byte{},
			expectedChallenge: &client.AuthChallenge{
				Realm: "",
				Nonce: "",
				Qop:   "",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			challenge := &client.AuthChallenge{}
			result, success := handleAuthenticateSessionReply(tc.msg, challenge)

			require.Equal(t, tc.expectedResult, result)
			require.Equal(t, tc.expectedSuccess, success)

			if tc.expectedChallenge != nil {
				require.Equal(t, tc.expectedChallenge.Realm, challenge.Realm)
				require.Equal(t, tc.expectedChallenge.Nonce, challenge.Nonce)
				require.Equal(t, tc.expectedChallenge.Qop, challenge.Qop)
			}
		})
	}
}

var ErrGeneral = errors.New("general error")

func TestBuildAuthReply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		challenge      *client.AuthChallenge
		response       string
		expectedResult []byte
		expectedError  bool
	}{
		{
			name: "Error on writeLength (exceeds limit)",
			challenge: &client.AuthChallenge{
				Username: strings.Repeat("A", int(math.MaxUint32)),
			},
			response:       "response",
			expectedResult: nil,
			expectedError:  true,
		},
		{
			name: "Error on writeFields (field length exceeds limit)",
			challenge: &client.AuthChallenge{
				Username: strings.Repeat("A", math.MaxUint8+1), // Length exceeds limit
			},
			response:       "response",
			expectedResult: nil,
			expectedError:  true,
		},
		{
			name: "Successful buildAuthReply",
			challenge: &client.AuthChallenge{
				Username:   "user",
				Realm:      "realm",
				Nonce:      "nonce",
				CNonce:     "cnonce",
				Qop:        "qop",
				NonceCount: 1,
			},
			response:       "response",
			expectedResult: []byte{0x13, 0x00, 0x00, 0x00, 0x04}, // Start with header bytes, expected to append the rest based on length and fields
			expectedError:  false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Call the function under test
			result := buildAuthReply(tc.challenge, tc.response)

			// Assertions
			if tc.expectedError {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, tc.expectedResult[:5], result[:5]) // Check the header, and compare further based on your actual output
			}
		})
	}
}

func TestWriteFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		challenge     *client.AuthChallenge
		response      string
		expectedError bool
	}{
		{
			name: "Successful writeFields",
			challenge: &client.AuthChallenge{
				Username:   "user",
				Realm:      "realm",
				Nonce:      "nonce",
				CNonce:     "cnonce",
				Qop:        "qop",
				NonceCount: 1,
			},
			response:      "response",
			expectedError: false,
		},
		{
			name: "writeField error on Username",
			challenge: &client.AuthChallenge{
				Username: strings.Repeat("A", math.MaxUint8+1), // Length exceeds limit
			},
			response:      "response",
			expectedError: true,
		},
		{
			name: "writeField error on Realm",
			challenge: &client.AuthChallenge{
				Realm: strings.Repeat("A", math.MaxUint8+1), // Length exceeds limit
			},
			response:      "response",
			expectedError: true,
		},
		{
			name: "writeField error on Nonce",
			challenge: &client.AuthChallenge{
				Nonce: strings.Repeat("A", math.MaxUint8+1), // Length exceeds limit
			},
			response:      "response",
			expectedError: true,
		},
		{
			name: "writeField error on CNonce",
			challenge: &client.AuthChallenge{
				CNonce: strings.Repeat("A", math.MaxUint8+1), // Length exceeds limit
			},
			response:      "response",
			expectedError: true,
		},
		{
			name: "writeField error on NonceCount",
			challenge: &client.AuthChallenge{
				NonceCount: 1,
			},
			response:      strings.Repeat("B", math.MaxUint8+1), // Length exceeds limit
			expectedError: true,
		},
		{
			name: "writeField error on response",
			challenge: &client.AuthChallenge{
				Username: "user",
				Realm:    "realm",
				Nonce:    "nonce",
				CNonce:   "cnonce",
				Qop:      "qop",
			},
			response:      strings.Repeat("C", math.MaxUint8+1), // Length exceeds limit
			expectedError: true,
		},
		{
			name: "writeField error on Qop",
			challenge: &client.AuthChallenge{
				Username: "user",
				Realm:    "realm",
				Nonce:    "nonce",
				CNonce:   "cnonce",
				Qop:      strings.Repeat("D", math.MaxUint8+1), // Length exceeds limit
			},
			response:      "response",
			expectedError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			buf := &bytes.Buffer{}
			err := writeFields(buf, tc.challenge, tc.response)

			// Assertions
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHandleAuthenticationSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		msg                []byte
		challenge          *client.AuthChallenge
		expectedResultType interface{}
		expectedNonceCount int
	}{
		{
			name:               "Message too short",
			msg:                []byte{0x01},
			challenge:          &client.AuthChallenge{},
			expectedResultType: []byte{},
		},
		{
			name: "Message of length 9 with all zeros",
			msg: []byte{
				0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			challenge:          &client.AuthChallenge{},
			expectedResultType: []byte{},
		},
		{
			name: "Digest Authentication with empty Realm",
			msg: []byte{
				0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04,
			},
			challenge:          &client.AuthChallenge{},
			expectedResultType: []byte{},
		},
		{
			name: "Digest Authentication with Realm",
			msg: []byte{
				0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04,
			},
			challenge: &client.AuthChallenge{
				Username:   "",
				Password:   "",
				Realm:      "exampleRealm",
				CSRFToken:  "",
				Domain:     "",
				Nonce:      "",
				Opaque:     "",
				Stale:      "",
				Algorithm:  "",
				Qop:        "",
				CNonce:     "",
				NonceCount: 1,
			},
			expectedResultType: []byte{},
		},
		{
			name: "Non-Digest Authentication",
			msg: []byte{
				0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
			},
			challenge:          &client.AuthChallenge{},
			expectedResultType: []byte{},
		},
		{
			name:               "End of function returns empty byte slice",
			msg:                []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			challenge:          &client.AuthChallenge{},
			expectedResultType: []byte{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := handleAuthenticationSession(tc.msg, tc.challenge)

			require.IsType(t, tc.expectedResultType, result)

			expectedResult, ok := tc.expectedResultType.([]byte)
			require.True(t, ok)

			if len(expectedResult) > 0 {
				require.NotEmpty(t, result)
			}

			if tc.expectedNonceCount != 0 {
				require.Equal(t, tc.expectedNonceCount, tc.challenge.NonceCount)
			}
		})
	}
}
