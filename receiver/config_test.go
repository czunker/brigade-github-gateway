package main

// nolint: lll
import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/brigadecore/brigade-foundations/http"
	"github.com/brigadecore/brigade-github-gateway/receiver/internal/webhooks"
	"github.com/brigadecore/brigade/sdk/v2/restmachinery"
	"github.com/stretchr/testify/require"
)

// Note that unit testing in Go does NOT clear environment variables between
// tests, which can sometimes be a pain, but it's fine here-- so each of these
// test functions uses a series of test cases that cumulatively build upon one
// another.

func TestAPIClientConfig(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func()
		assertions func(
			address string,
			token string,
			opts restmachinery.APIClientOptions,
			err error,
		)
	}{
		{
			name:  "API_ADDRESS not set",
			setup: func() {},
			assertions: func(
				_ string,
				_ string,
				_ restmachinery.APIClientOptions,
				err error,
			) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "value not found for")
				require.Contains(t, err.Error(), "API_ADDRESS")
			},
		},
		{
			name: "API_TOKEN not set",
			setup: func() {
				t.Setenv("API_ADDRESS", "foo")
			},
			assertions: func(
				_ string,
				_ string,
				_ restmachinery.APIClientOptions,
				err error,
			) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "value not found for")
				require.Contains(t, err.Error(), "API_TOKEN")
			},
		},
		{
			name: "SUCCESS not set",
			setup: func() {
				t.Setenv("API_TOKEN", "bar")
				t.Setenv("API_IGNORE_CERT_WARNINGS", "true")
			},
			assertions: func(
				address string,
				token string,
				opts restmachinery.APIClientOptions,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, "foo", address)
				require.Equal(t, "bar", token)
				require.True(t, opts.AllowInsecureConnections)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.setup()
			address, token, opts, err := apiClientConfig()
			testCase.assertions(address, token, opts, err)
		})
	}
}

func TestWebHookServiceConfig(t *testing.T) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write([]byte("foo"))
	require.NoError(t, err)
	testCases := []struct {
		name       string
		setup      func()
		assertions func(webhooks.ServiceConfig, error)
	}{
		{
			name: "GITHUB_APPS_PATH not set",
			assertions: func(_ webhooks.ServiceConfig, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "value not found for")
				require.Contains(t, err.Error(), "GITHUB_APPS_PATH")
			},
		},
		{
			name: "GITHUB_APPS_PATH path does not exist",
			setup: func() {
				t.Setenv("GITHUB_APPS_PATH", "/completely/bogus/path")
			},
			assertions: func(_ webhooks.ServiceConfig, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"file /completely/bogus/path does not exist",
				)
			},
		},
		{
			name: "GITHUB_APPS_PATH does not contain valid json",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err = appsFile.Write([]byte("this is not json"))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
			},
			assertions: func(_ webhooks.ServiceConfig, err error) {
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "invalid character",
				)
			},
		},
		{
			name: "CHECK_SUITE_ALLOWED_AUTHOR_ASSOCIATIONS not defined",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err =
					appsFile.Write([]byte(`[{"appID":42,"sharedSecret":"foobar"}]`))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
			},
			assertions: func(config webhooks.ServiceConfig, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]string{},
					config.CheckSuiteAllowedAuthorAssociations,
				)
			},
		},
		{
			name: "CHECK_SUITE_ALLOWED_AUTHOR_ASSOCIATIONS defined",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err =
					appsFile.Write([]byte(`[{"appID":42,"sharedSecret":"foobar"}]`))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
				t.Setenv("CHECK_SUITE_ALLOWED_AUTHOR_ASSOCIATIONS", "FOO,BAR")
			},
			assertions: func(config webhooks.ServiceConfig, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]string{"FOO", "BAR"},
					config.CheckSuiteAllowedAuthorAssociations,
				)
			},
		},
		{
			name: "EMITTED_EVENTS not defined",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err =
					appsFile.Write([]byte(`[{"appID":42,"sharedSecret":"foobar"}]`))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
			},
			assertions: func(config webhooks.ServiceConfig, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]string{"*"},
					config.EmittedEvents,
				)
			},
		},
		{
			name: "EMITTED_EVENTS defined",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err =
					appsFile.Write([]byte(`[{"appID":42,"sharedSecret":"foobar"}]`))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
				t.Setenv("EMITTED_EVENTS", "foo,bar")
			},
			assertions: func(config webhooks.ServiceConfig, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]string{"foo", "bar"},
					config.EmittedEvents,
				)
			},
		},
		{
			name: "CHECK_SUITE_ON_PR not defined",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err =
					appsFile.Write([]byte(`[{"appID":42,"sharedSecret":"foobar"}]`))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
			},
			assertions: func(config webhooks.ServiceConfig, err error) {
				require.NoError(t, err)
				require.True(t, config.CheckSuiteOnPR)
			},
		},
		{
			name: "CHECK_SUITE_ON_PR not a bool",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err =
					appsFile.Write([]byte(`[{"appID":42,"sharedSecret":"foobar"}]`))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
				t.Setenv("CHECK_SUITE_ON_PR", "i am not a bool")
			},
			assertions: func(_ webhooks.ServiceConfig, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "was not parsable as a bool")
			},
		},
		{
			name: "CHECK_SUITE_ON_PR defined correctly",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err =
					appsFile.Write([]byte(`[{"appID":42,"sharedSecret":"foobar"}]`))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
				t.Setenv("CHECK_SUITE_ON_PR", "false")
			},
			assertions: func(config webhooks.ServiceConfig, err error) {
				require.NoError(t, err)
				require.False(t, config.CheckSuiteOnPR)
			},
		},
		{
			name: "CHECK_SUITE_ON_COMMENT not defined",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err =
					appsFile.Write([]byte(`[{"appID":42,"sharedSecret":"foobar"}]`))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
			},
			assertions: func(config webhooks.ServiceConfig, err error) {
				require.NoError(t, err)
				require.True(t, config.CheckSuiteOnComment)
			},
		},
		{
			name: "CHECK_SUITE_ON_COMMENT not a bool",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err =
					appsFile.Write([]byte(`[{"appID":42,"sharedSecret":"foobar"}]`))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
				t.Setenv("CHECK_SUITE_ON_COMMENT", "i am not a bool")
			},
			assertions: func(_ webhooks.ServiceConfig, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "was not parsable as a bool")
			},
		},
		{
			name: "CHECK_SUITE_ON_COMMENT defined correctly",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err =
					appsFile.Write([]byte(`[{"appID":42,"sharedSecret":"foobar"}]`))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
				t.Setenv("CHECK_SUITE_ON_COMMENT", "false")
			},
			assertions: func(config webhooks.ServiceConfig, err error) {
				require.NoError(t, err)
				require.False(t, config.CheckSuiteOnComment)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.setup != nil {
				testCase.setup()
			}
			testCase.assertions(webhookServiceConfig())
		})
	}
}

func TestSignatureVerificationFilterConfig(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func()
		assertions func(webhooks.SignatureVerificationFilterConfig, error)
	}{
		{
			name: "GITHUB_APPS_PATH not set",
			assertions: func(
				_ webhooks.SignatureVerificationFilterConfig,
				err error,
			) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "value not found for")
				require.Contains(t, err.Error(), "GITHUB_APPS_PATH")
			},
		},
		{
			name: "GITHUB_APPS_PATH path does not exist",
			setup: func() {
				t.Setenv("GITHUB_APPS_PATH", "/completely/bogus/path")
			},
			assertions: func(
				_ webhooks.SignatureVerificationFilterConfig,
				err error,
			) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"file /completely/bogus/path does not exist",
				)
			},
		},
		{
			name: "GITHUB_APPS_PATH does not contain valid json",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err = appsFile.Write([]byte("this is not json"))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
			},
			assertions: func(
				_ webhooks.SignatureVerificationFilterConfig,
				err error,
			) {
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "invalid character",
				)
			},
		},
		{
			name: "success",
			setup: func() {
				appsFile, err := ioutil.TempFile("", "apps.json")
				require.NoError(t, err)
				defer appsFile.Close()
				_, err =
					appsFile.Write([]byte(`[{"appID":42,"sharedSecret":"foobar"}]`))
				require.NoError(t, err)
				t.Setenv("GITHUB_APPS_PATH", appsFile.Name())
			},
			assertions: func(
				config webhooks.SignatureVerificationFilterConfig,
				err error,
			) {
				require.NoError(t, err)
				require.Len(t, config.GitHubApps, 1)
				require.Equal(t, int64(42), config.GitHubApps[42].AppID)
				require.Equal(t, "foobar", config.GitHubApps[42].SharedSecret)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.setup != nil {
				testCase.setup()
			}
			config, err := signatureVerificationFilterConfig()
			testCase.assertions(config, err)
		})
	}
}

func TestServerConfig(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func()
		assertions func(http.ServerConfig, error)
	}{
		{
			name: "RECEIVER_PORT not an int",
			setup: func() {
				t.Setenv("RECEIVER_PORT", "foo")
			},
			assertions: func(_ http.ServerConfig, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "was not parsable as an int")
				require.Contains(t, err.Error(), "RECEIVER_PORT")
			},
		},
		{
			name: "TLS_ENABLED not a bool",
			setup: func() {
				t.Setenv("RECEIVER_PORT", "8080")
				t.Setenv("TLS_ENABLED", "nope")
			},
			assertions: func(_ http.ServerConfig, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "was not parsable as a bool")
				require.Contains(t, err.Error(), "TLS_ENABLED")
			},
		},
		{
			name: "TLS_CERT_PATH required but not set",
			setup: func() {
				t.Setenv("TLS_ENABLED", "true")
			},
			assertions: func(_ http.ServerConfig, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "value not found for")
				require.Contains(t, err.Error(), "TLS_CERT_PATH")
			},
		},
		{
			name: "TLS_KEY_PATH required but not set",
			setup: func() {
				t.Setenv("TLS_CERT_PATH", "/var/ssl/cert")
			},
			assertions: func(_ http.ServerConfig, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "value not found for")
				require.Contains(t, err.Error(), "TLS_KEY_PATH")
			},
		},
		{
			name: "success",
			setup: func() {
				t.Setenv("TLS_KEY_PATH", "/var/ssl/key")
			},
			assertions: func(config http.ServerConfig, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					http.ServerConfig{
						Port:        8080,
						TLSEnabled:  true,
						TLSCertPath: "/var/ssl/cert",
						TLSKeyPath:  "/var/ssl/key",
					},
					config,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.setup()
			config, err := serverConfig()
			testCase.assertions(config, err)
		})
	}
}
