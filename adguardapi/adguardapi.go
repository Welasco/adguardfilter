package adguardapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	logger "github.com/welasco/adguardfilter/common/logger"
	"github.com/welasco/adguardfilter/model"
)

// GetBlockedServices retrieves the blocked services configuration from the API
func GetBlockedServices() (model.ServiceConfig, error) {
	// Create the GET request
	apiURL := authBaseURL + "/control/blocked_services/get"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		logger.Error("[adguardapi][GetBlockedServices] Failed to create GET request")
		logger.Error(err)
		return model.ServiceConfig{}, err
	}

	// Set required headers
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Perform the authenticated request (will auto re-authenticate if needed)
	resp, err := DoAuthenticatedRequest(req)
	if err != nil {
		logger.Error("[adguardapi][GetBlockedServices] Failed to get blocked services from: " + apiURL)
		logger.Error(err)
		return model.ServiceConfig{}, err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != 200 {
		logger.Error("[adguardapi][GetBlockedServices] Request failed with status: " + resp.Status)
		return model.ServiceConfig{}, errors.New("request failed with status: " + resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[adguardapi][GetBlockedServices] Failed to read response body")
		logger.Error(err)
		return model.ServiceConfig{}, err
	}

	logger.Debug("[adguardapi][GetBlockedServices] Response body: " + string(body))

	// Unmarshal JSON into ServiceConfig model
	var serviceConfig model.ServiceConfig
	err = json.Unmarshal(body, &serviceConfig)
	if err != nil {
		logger.Error("[adguardapi][GetBlockedServices] Failed to unmarshal JSON response")
		logger.Error(err)
		return model.ServiceConfig{}, err
	}

	logger.Info("[adguardapi][GetBlockedServices] Successfully retrieved blocked services configuration")
	logger.Debug("[adguardapi][GetBlockedServices] Number of blocked service IDs: ", len(serviceConfig.IDs))
	logger.Debug("[adguardapi][GetBlockedServices] Timezone: " + serviceConfig.Schedule.TimeZone)

	return serviceConfig, nil
}

// GetAllBlockedServices retrieves all available blocked services from the API
func GetAllBlockedServices() ([]model.BlockedService, error) {
	apiURL := authBaseURL + "/control/blocked_services/all"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		logger.Error("[adguardapi][GetAllBlockedServices] Failed to create GET request")
		logger.Error(err)
		return nil, err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := DoAuthenticatedRequest(req)
	if err != nil {
		logger.Error("[adguardapi][GetAllBlockedServices] Failed to get all blocked services from: " + apiURL)
		logger.Error(err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Error("[adguardapi][GetAllBlockedServices] Request failed with status: " + resp.Status)
		return nil, errors.New("request failed with status: " + resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[adguardapi][GetAllBlockedServices] Failed to read response body")
		logger.Error(err)
		return nil, err
	}

	logger.Debug("[adguardapi][GetAllBlockedServices] Response body: " + string(body))

	var allServicesResp model.AllBlockedServicesResponse
	err = json.Unmarshal(body, &allServicesResp)
	if err != nil {
		logger.Error("[adguardapi][GetAllBlockedServices] Failed to unmarshal JSON response")
		logger.Error(err)
		return nil, err
	}

	logger.Info("[adguardapi][GetAllBlockedServices] Successfully retrieved all blocked services")
	logger.Debug("[adguardapi][GetAllBlockedServices] Number of services: ", len(allServicesResp.BlockedServices))

	return allServicesResp.BlockedServices, nil
}

// UpdateBlockedServices updates the blocked services configuration via the API
func UpdateBlockedServices(serviceConfig *model.ServiceConfig) error {

	// Marshal the ServiceConfig to JSON
	jsonData, err := json.Marshal(serviceConfig)
	if err != nil {
		logger.Error("[adguardapi][UpdateBlockedServices] Failed to marshal ServiceConfig to JSON")
		logger.Error(err)
		return err
	}
	logger.Debug("[adguardapi][UpdateBlockedServices] Request body: " + string(jsonData))

	// Create the PUT request
	apiURL := authBaseURL + "/control/blocked_services/update"
	req, err := http.NewRequest("PUT", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("[adguardapi][UpdateBlockedServices] Failed to create PUT request")
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
		logger.Error("[adguardapi][UpdateBlockedServices] Failed to update blocked services to: " + apiURL)
		logger.Error(err)
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != 200 {
		// Read error response body for better debugging
		body, _ := io.ReadAll(resp.Body)
		logger.Error("[adguardapi][UpdateBlockedServices] Request failed with status: " + resp.Status)
		logger.Error("[adguardapi][UpdateBlockedServices] Response body: " + string(body))
		return errors.New("request failed with status: " + resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[adguardapi][UpdateBlockedServices] Failed to read response body")
		logger.Error(err)
		return err
	}

	logger.Debug("[adguardapi][UpdateBlockedServices] Response body: " + string(body))
	logger.Info("[adguardapi][UpdateBlockedServices] Successfully updated blocked services configuration")

	return nil
}

// ResetBlockedServices resets all blocked services to the default list
func ResetBlockedServices() error {
	// Default blocked service IDs
	defaultIDs := []string{
		"tinder", "plenty_of_fish", "onlyfans", "playstation", "nintendo", "tiktok",
		"aliexpress", "500px", "activision_blizzard", "battle_net", "betway", "blaze",
		"box", "crunchyroll", "directvgo", "disneyplus", "ebay", "espn", "flickr",
		"iheartradio", "iqiyi", "kook", "line", "mercado_libre", "ok", "origin", "qq",
		"riot_games", "signal", "tidal", "tumblr", "ubisoft", "vimeo", "wargaming",
		"xiaohongshu", "zhihu", "yy", "weibo", "wechat", "voot", "viber", "twitch",
		"wizz", "shein", "paramountplus", "pluto_tv", "mail_ru", "kakaotalk", "imgur",
		"hulu", "globoplay", "dailymotion", "clubhouse", "canais_globo", "betano",
		"bigo_live", "amino", "9gag", "betfair", "bilibili", "bluesky", "claro",
		"coolapk", "deezer", "kik", "leagueoflegends", "lionsgateplus", "mastodon",
		"rockstar_games", "temu", "telegram", "soundcloud", "samsung_tv_plus", "looke",
		"hbomax", "discoveryplus", "gog", "nebula", "facebook", "privacy", "snapchat",
		"youtube", "roblox", "spotify_video", "spotify",
	}

	// Override with env var if set (comma-separated list of service IDs)
	if envIDs := os.Getenv("defaultBlockedServices"); envIDs != "" {
		logger.Info("[adguardapi][ResetBlockedServices] Loading default blocked services from environment variable")
		defaultIDs = strings.Split(envIDs, ",")
		for i := range defaultIDs {
			defaultIDs[i] = strings.TrimSpace(defaultIDs[i])
		}
	}

	defaultConfig := model.ServiceConfig{
		Schedule: model.Schedule{
			TimeZone: "America/Chicago",
		},
		IDs: defaultIDs,
	}

	logger.Info("[adguardapi][ResetBlockedServices] Resetting blocked services to default configuration")
	logger.Debug("[adguardapi][ResetBlockedServices] Default service count: ", len(defaultConfig.IDs))

	// Marshal the ServiceConfig to JSON
	jsonData, err := json.Marshal(defaultConfig)
	if err != nil {
		logger.Error("[adguardapi][ResetBlockedServices] Failed to marshal default ServiceConfig to JSON")
		logger.Error(err)
		return err
	}

	logger.Debug("[adguardapi][ResetBlockedServices] Request body: " + string(jsonData))

	// Create the PUT request
	apiURL := authBaseURL + "/control/blocked_services/update"
	req, err := http.NewRequest("PUT", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("[adguardapi][ResetBlockedServices] Failed to create PUT request")
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
		logger.Error("[adguardapi][ResetBlockedServices] Failed to reset blocked services to: " + apiURL)
		logger.Error(err)
		return err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != 200 {
		// Read error response body for better debugging
		body, _ := io.ReadAll(resp.Body)
		logger.Error("[adguardapi][ResetBlockedServices] Request failed with status: " + resp.Status)
		logger.Error("[adguardapi][ResetBlockedServices] Response body: " + string(body))
		return errors.New("request failed with status: " + resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[adguardapi][ResetBlockedServices] Failed to read response body")
		logger.Error(err)
		return err
	}

	logger.Debug("[adguardapi][ResetBlockedServices] Response body: " + string(body))
	logger.Info("[adguardapi][ResetBlockedServices] Successfully reset blocked services to default configuration")

	return nil
}
