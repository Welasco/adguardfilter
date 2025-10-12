package adguardapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	logger "github.com/welasco/adguardfilter/common/logger"
)

// AuthCredentials holds the credentials for API authentication
type AuthCredentials struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

var (
	// Global HTTP client with cookie jar for session management
	httpClient *http.Client
	// Store credentials for automatic re-authentication
	authBaseURL  string
	authUsername string
	authPassword string
)

func init() {
	// Initialize variables if needed
	authBaseURL = os.Getenv("authBaseURL")
	authUsername = os.Getenv("authUsername")
	authPassword = os.Getenv("authPassword")
}

// InitHTTPClient initializes the HTTP client with a cookie jar
func InitHTTPClient() error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		logger.Error("[adguardapi_auth][InitHTTPClient] Failed to create cookie jar")
		logger.Error(err)
		return err
	}

	httpClient = &http.Client{
		Jar: jar,
	}

	logger.Debug("[adguardapi_auth][InitHTTPClient] HTTP client initialized with cookie jar")
	return nil
}

// Authenticate performs authentication against the API and stores the session cookie
func Authenticate(baseURL, username, password string) error {
	// Initialize HTTP client if not already done
	if httpClient == nil {
		if err := InitHTTPClient(); err != nil {
			return err
		}
	}

	// Prepare authentication credentials
	credentials := AuthCredentials{
		Name:     username,
		Password: password,
	}

	jsonData, err := json.Marshal(credentials)
	if err != nil {
		logger.Error("[adguardapi_auth][Authenticate] Failed to marshal credentials")
		logger.Error(err)
		return err
	}

	// Create the authentication request
	loginURL := baseURL + "/control/login"
	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("[adguardapi_auth][Authenticate] Failed to create authentication request")
		logger.Error(err)
		return err
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Perform the request
	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Error("[adguardapi_auth][Authenticate] Failed to authenticate to: " + loginURL)
		logger.Error(err)
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != 200 {
		logger.Error("[adguardapi_auth][Authenticate] Authentication failed with status: " + resp.Status)
		return errors.New("authentication failed with status: " + resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[adguardapi_auth][Authenticate] Failed to read response body")
		logger.Error(err)
		return err
	}

	logger.Debug("[adguardapi_auth][Authenticate] Authentication response: " + string(body))

	// Verify cookies were set
	parsedURL, _ := url.Parse(baseURL)
	cookies := httpClient.Jar.Cookies(parsedURL)
	if len(cookies) == 0 {
		logger.Error("[adguardapi_auth][Authenticate] No cookies received from authentication")
		return errors.New("no cookies received from authentication")
	}

	logger.Info("[adguardapi_auth][Authenticate] Successfully authenticated. Cookies stored for future requests")
	for _, cookie := range cookies {
		logger.Debug("[adguardapi_auth][Authenticate] Cookie: " + cookie.Name + "=" + cookie.Value)
	}

	// Store credentials for automatic re-authentication
	authBaseURL = baseURL
	authUsername = username
	authPassword = password

	return nil
}

// GetHTTPClient returns the global HTTP client with cookies
func GetHTTPClient() (*http.Client, error) {
	if httpClient == nil {
		return nil, errors.New("HTTP client not initialized. Call InitHTTPClient or Authenticate first")
	}
	return httpClient, nil
}

// isAuthError checks if the status code indicates an authentication/authorization error
func isAuthError(statusCode int) bool {
	return statusCode == 401 || statusCode == 403
}

// canReauthenticate checks if we have stored credentials for re-authentication
func canReauthenticate() bool {
	return authBaseURL != "" && authUsername != "" && authPassword != ""
}

// DoAuthenticatedRequest performs an HTTP request with automatic re-authentication on auth failures
func DoAuthenticatedRequest(req *http.Request) (*http.Response, error) {
	// Ensure HTTP client is initialized
	if httpClient == nil {
		if err := InitHTTPClient(); err != nil {
			return nil, err
		}
	}

	// Perform the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Check if we got an authentication error
	if isAuthError(resp.StatusCode) {
		resp.Body.Close() // Close the failed response body

		// Check if we can re-authenticate
		if !canReauthenticate() {
			logger.Error("[adguardapi_auth][DoAuthenticatedRequest] Authentication failed and no credentials stored for re-authentication")
			return nil, errors.New("authentication required but no credentials available")
		}

		logger.Info("[adguardapi_auth][DoAuthenticatedRequest] Session expired (status " + resp.Status + "), attempting re-authentication...")

		// Attempt to re-authenticate
		if err := Authenticate(authBaseURL, authUsername, authPassword); err != nil {
			logger.Error("[adguardapi_auth][DoAuthenticatedRequest] Re-authentication failed")
			return nil, err
		}

		logger.Info("[adguardapi_auth][DoAuthenticatedRequest] Re-authentication successful, retrying original request")

		// Retry the original request with new cookies
		resp, err = httpClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}
