package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"

	"github.com/gofiber/fiber/v2"
	logger "github.com/welasco/adguardfilter/common/logger"
	"github.com/welasco/adguardfilter/model"
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
		logger.Error("[http][InitHTTPClient] Failed to create cookie jar")
		logger.Error(err)
		return err
	}

	httpClient = &http.Client{
		Jar: jar,
	}

	logger.Debug("[http][InitHTTPClient] HTTP client initialized with cookie jar")
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
		logger.Error("[http][Authenticate] Failed to marshal credentials")
		logger.Error(err)
		return err
	}

	// Create the authentication request
	loginURL := baseURL + "/control/login"
	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("[http][Authenticate] Failed to create authentication request")
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
		logger.Error("[http][Authenticate] Failed to authenticate to: " + loginURL)
		logger.Error(err)
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != 200 {
		logger.Error("[http][Authenticate] Authentication failed with status: " + resp.Status)
		return errors.New("authentication failed with status: " + resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[http][Authenticate] Failed to read response body")
		logger.Error(err)
		return err
	}

	logger.Debug("[http][Authenticate] Authentication response: " + string(body))

	// Verify cookies were set
	parsedURL, _ := url.Parse(baseURL)
	cookies := httpClient.Jar.Cookies(parsedURL)
	if len(cookies) == 0 {
		logger.Error("[http][Authenticate] No cookies received from authentication")
		return errors.New("no cookies received from authentication")
	}

	logger.Info("[http][Authenticate] Successfully authenticated. Cookies stored for future requests")
	for _, cookie := range cookies {
		logger.Debug("[http][Authenticate] Cookie: " + cookie.Name + "=" + cookie.Value)
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
			logger.Error("[http][DoAuthenticatedRequest] Authentication failed and no credentials stored for re-authentication")
			return nil, errors.New("authentication required but no credentials available")
		}

		logger.Info("[http][DoAuthenticatedRequest] Session expired (status " + resp.Status + "), attempting re-authentication...")

		// Attempt to re-authenticate
		if err := Authenticate(authBaseURL, authUsername, authPassword); err != nil {
			logger.Error("[http][DoAuthenticatedRequest] Re-authentication failed")
			return nil, err
		}

		logger.Info("[http][DoAuthenticatedRequest] Re-authentication successful, retrying original request")

		// Retry the original request with new cookies
		resp, err = httpClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// GetBlockedServices retrieves the blocked services configuration from the API
func GetBlockedServices(c *fiber.Ctx) error {
	// Create the GET request
	apiURL := authBaseURL + "/control/blocked_services/get"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		logger.Error("[http][GetBlockedServices] Failed to create GET request")
		logger.Error(err)
		return err
	}

	// Set required headers
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Perform the authenticated request (will auto re-authenticate if needed)
	resp, err := DoAuthenticatedRequest(req)
	if err != nil {
		logger.Error("[http][GetBlockedServices] Failed to get blocked services from: " + apiURL)
		logger.Error(err)
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != 200 {
		logger.Error("[http][GetBlockedServices] Request failed with status: " + resp.Status)
		return errors.New("request failed with status: " + resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[http][GetBlockedServices] Failed to read response body")
		logger.Error(err)
		return err
	}

	logger.Debug("[http][GetBlockedServices] Response body: " + string(body))

	// Unmarshal JSON into ServiceConfig model
	var serviceConfig model.ServiceConfig
	err = json.Unmarshal(body, &serviceConfig)
	if err != nil {
		logger.Error("[http][GetBlockedServices] Failed to unmarshal JSON response")
		logger.Error(err)
		return err
	}

	logger.Info("[http][GetBlockedServices] Successfully retrieved blocked services configuration")
	logger.Debug("[http][GetBlockedServices] Number of blocked service IDs: ", len(serviceConfig.IDs))
	logger.Debug("[http][GetBlockedServices] Timezone: " + serviceConfig.Schedule.TimeZone)

	return c.JSON(&serviceConfig)
}

// UpdateBlockedServices updates the blocked services configuration via the API
func UpdateBlockedServices(c *fiber.Ctx) error {

	// Marshal the ServiceConfig to JSON
	//jsonData, err := json.Marshal(config)
	var serviceConfig model.ServiceConfig
	err := c.BodyParser(&serviceConfig)
	if err != nil {
		logger.Error("[http][UpdateBlockedServices] Failed to marshal ServiceConfig")
		logger.Error(err)
		return err
	}

	jsonData, err := json.Marshal(serviceConfig)
	if err != nil {
		logger.Error("[http][UpdateBlockedServices] Failed to marshal ServiceConfig to JSON")
		logger.Error(err)
		return err
	}
	logger.Debug("[http][UpdateBlockedServices] Request body: " + string(jsonData))

	// Create the PUT request
	apiURL := authBaseURL + "/control/blocked_services/update"
	req, err := http.NewRequest("PUT", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("[http][UpdateBlockedServices] Failed to create PUT request")
		logger.Error(err)
		return err
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Perform the authenticated request (will auto re-authenticate if needed)
	resp, err := DoAuthenticatedRequest(req)
	if err != nil {
		logger.Error("[http][UpdateBlockedServices] Failed to update blocked services to: " + apiURL)
		logger.Error(err)
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != 200 {
		// Read error response body for better debugging
		body, _ := io.ReadAll(resp.Body)
		logger.Error("[http][UpdateBlockedServices] Request failed with status: " + resp.Status)
		logger.Error("[http][UpdateBlockedServices] Response body: " + string(body))
		return errors.New("request failed with status: " + resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[http][UpdateBlockedServices] Failed to read response body")
		logger.Error(err)
		return err
	}

	logger.Debug("[http][UpdateBlockedServices] Response body: " + string(body))
	logger.Info("[http][UpdateBlockedServices] Successfully updated blocked services configuration")

	return c.SendStatus(fiber.StatusOK)
}
