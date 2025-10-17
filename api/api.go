package api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/welasco/adguardfilter/adguardapi"
	logger "github.com/welasco/adguardfilter/common/logger"
	"github.com/welasco/adguardfilter/common/servicelist"
	"github.com/welasco/adguardfilter/common/timer"
	"github.com/welasco/adguardfilter/model"
)

// ApiGetServiceList retrieves the list of available services from the API
func ApiGetServiceList(c *fiber.Ctx) error {
	// Retrieve the service list via the adguardapi package
	var serviceList = servicelist.GetBlockedServices()

	logger.Info("[adguardapi][ApiGetServiceList] Successfully retrieved service list")
	return c.JSON(&serviceList)
}

// ApiGetBlockedServices retrieves the blocked services configuration from the API
func ApiGetBlockedServices(c *fiber.Ctx) error {

	// Unmarshal JSON into ServiceConfig model
	serviceConfig, err := adguardapi.GetBlockedServices()
	if err != nil {
		logger.Error("[adguardapi][ApiGetBlockedServices] Failed to get blocked services")
		logger.Error(err)
		return err
	}

	logger.Info("[adguardapi][ApiGetBlockedServices] Successfully retrieved blocked services configuration")
	logger.Debug("[adguardapi][ApiGetBlockedServices] Number of blocked service IDs: ", len(serviceConfig.IDs))
	logger.Debug("[adguardapi][ApiGetBlockedServices] Timezone: " + serviceConfig.Schedule.TimeZone)

	return c.JSON(&serviceConfig)
}

// ApiUpdateBlockedServicesMin updates the blocked services configuration via the API
func ApiUpdateBlockedServicesMin(c *fiber.Ctx) error {
	// Parse request body into ResetServiceMinConfig model
	var resetServiceConfig model.ResetServiceMinConfig
	err := c.BodyParser(&resetServiceConfig)
	if err != nil {
		logger.Error("[api][ApiUpdateBlockedServicesMin] Failed to parse request body")
		logger.Error(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to parse request body",
		})
	}

	// Update blocked services via the API
	err = adguardapi.UpdateBlockedServices(&resetServiceConfig.ServiceConfig)
	if err != nil {
		logger.Error("[api][ApiUpdateBlockedServicesMin] Failed to update blocked services")
		logger.Error(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update blocked services",
		})
	}

	logger.Info("[api][ApiUpdateBlockedServicesMin] Successfully updated blocked services")

	// If a reset duration is specified, set a timer to reset the configuration
	if resetServiceConfig.ResetAfterMin > 0 {
		// Stop all existing timers to ensure only one timer is active
		activeTimers := timer.GetAllActiveTimers()
		if len(activeTimers) > 0 {
			logger.Info("[api][ApiUpdateBlockedServicesMin] Stopping ", len(activeTimers), " existing timer(s) before creating new one")
			timer.StopAllTimers()
			logger.Info("[api][ApiUpdateBlockedServicesMin] All existing timers stopped")
		}

		timerID := fmt.Sprintf("reset-blocked-services-%d", resetServiceConfig.ResetAfterMin)

		logger.Info("[api][ApiUpdateBlockedServicesMin] Creating timer to reset blocked services after ", resetServiceConfig.ResetAfterMin, " minutes")

		// Create a timer with the ResetBlockedServices callback
		_, err := timer.NewTimerWithDuration(timerID, resetServiceConfig.ResetAfterMin, func() {
			logger.Info("[api][Timer Callback] Resetting blocked services to default configuration")
			err := adguardapi.ResetBlockedServices()
			if err != nil {
				logger.Error("[api][Timer Callback] Failed to reset blocked services")
				logger.Error(err)
			} else {
				logger.Info("[api][Timer Callback] Successfully reset blocked services to default")
			}
		})

		if err != nil {
			logger.Error("[api][ApiUpdateBlockedServicesMin] Failed to create timer")
			logger.Error(err)
			// Don't fail the request, just log the error
			return c.JSON(fiber.Map{
				"success":     true,
				"message":     "Blocked services updated, but timer creation failed",
				"timer_error": err.Error(),
			})
		}

		logger.Info("[api][ApiUpdateBlockedServicesMin] Timer created successfully with ID: " + timerID)

		return c.JSON(fiber.Map{
			"success":         true,
			"message":         fmt.Sprintf("Blocked services updated and will reset to default in %d minutes", resetServiceConfig.ResetAfterMin),
			"timer_id":        timerID,
			"reset_after_min": resetServiceConfig.ResetAfterMin,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Blocked services updated (no reset timer set)",
	})
}

// ApiUpdateBlockedServicesDateTime updates the blocked services configuration and sets a timer with a specific deadline
func ApiUpdateBlockedServicesDateTime(c *fiber.Ctx) error {
	// Parse request body into ResetServiceDateTimeConfig model
	var resetServiceConfig model.ResetServiceDateTimeConfig
	err := c.BodyParser(&resetServiceConfig)
	if err != nil {
		logger.Error("[api][ApiUpdateBlockedServicesDateTime] Failed to parse request body")
		logger.Error(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to parse request body",
		})
	}

	// Validate that reset_date_time is provided
	if resetServiceConfig.ResetDateTime == "" {
		logger.Error("[api][ApiUpdateBlockedServicesDateTime] reset_date_time is required")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "reset_date_time is required",
		})
	}

	// Parse the datetime string (supports multiple formats from JavaScript)
	var deadline time.Time

	// Try parsing common datetime formats from JavaScript
	formats := []string{
		time.RFC3339,                // "2025-10-12T15:30:00Z"
		"2006-01-02T15:04:05Z07:00", // ISO 8601 with timezone
		"2006-01-02T15:04:05",       // ISO 8601 without timezone
		"2006-01-02 15:04:05",       // Common format
		time.RFC3339Nano,            // With nanoseconds
	}

	parseError := fmt.Errorf("invalid datetime format")
	for _, format := range formats {
		deadline, err = time.Parse(format, resetServiceConfig.ResetDateTime)
		if err == nil {
			parseError = nil
			break
		}
	}

	if parseError != nil {
		logger.Error("[api][ApiUpdateBlockedServicesDateTime] Failed to parse reset_date_time: " + resetServiceConfig.ResetDateTime)
		logger.Error(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid datetime format. Use ISO 8601 format (e.g., 2025-10-12T15:30:00Z)",
			"example": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		})
	}

	// Check if the deadline is in the future
	if deadline.Before(time.Now()) {
		logger.Error("[api][ApiUpdateBlockedServicesDateTime] Deadline is in the past: " + deadline.Format(time.RFC3339))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":           "Deadline must be in the future",
			"provided_time":   deadline.Format(time.RFC3339),
			"current_time":    time.Now().Format(time.RFC3339),
			"time_difference": time.Until(deadline).String(),
		})
	}

	// Update blocked services via the API
	err = adguardapi.UpdateBlockedServices(&resetServiceConfig.ServiceConfig)
	if err != nil {
		logger.Error("[api][ApiUpdateBlockedServicesDateTime] Failed to update blocked services")
		logger.Error(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update blocked services",
		})
	}

	logger.Info("[api][ApiUpdateBlockedServicesDateTime] Successfully updated blocked services")

	// Stop all existing timers to ensure only one timer is active
	activeTimers := timer.GetAllActiveTimers()
	if len(activeTimers) > 0 {
		logger.Info("[api][ApiUpdateBlockedServicesDateTime] Stopping ", len(activeTimers), " existing timer(s) before creating new one")
		timer.StopAllTimers()
		logger.Info("[api][ApiUpdateBlockedServicesDateTime] All existing timers stopped")
	}

	// Create a timer with the deadline
	timerID := fmt.Sprintf("reset-blocked-services-%d", time.Now().Unix())
	durationUntilReset := time.Until(deadline)

	logger.Info("[api][ApiUpdateBlockedServicesDateTime] Creating timer to reset blocked services at: " + deadline.Format(time.RFC3339))
	logger.Debug("[api][ApiUpdateBlockedServicesDateTime] Duration until reset: " + durationUntilReset.String())

	// Create a timer with the ResetBlockedServices callback
	_, err = timer.NewTimerWithDeadline(timerID, deadline, func() {
		logger.Info("[api][Timer Callback] Resetting blocked services to default configuration (scheduled deadline reached)")
		err := adguardapi.ResetBlockedServices()
		if err != nil {
			logger.Error("[api][Timer Callback] Failed to reset blocked services")
			logger.Error(err)
		} else {
			logger.Info("[api][Timer Callback] Successfully reset blocked services to default")
		}
	})

	if err != nil {
		logger.Error("[api][ApiUpdateBlockedServicesDateTime] Failed to create timer")
		logger.Error(err)
		// Don't fail the request, just log the error
		return c.JSON(fiber.Map{
			"success":     true,
			"message":     "Blocked services updated, but timer creation failed",
			"timer_error": err.Error(),
		})
	}

	logger.Info("[api][ApiUpdateBlockedServicesDateTime] Timer created successfully with ID: " + timerID)

	return c.JSON(fiber.Map{
		"success":          true,
		"message":          "Blocked services updated and will reset to default at specified time",
		"timer_id":         timerID,
		"reset_date_time":  deadline.Format(time.RFC3339),
		"time_until_reset": durationUntilReset.String(),
		"current_time":     time.Now().Format(time.RFC3339),
	})
}

// ApiGetTimer retrieves information about the currently active timer
func ApiGetTimer(c *fiber.Ctx) error {
	// Get all active timers (should be at most one)
	activeTimers := timer.GetAllActiveTimers()

	logger.Debug("[api][ApiGetTimer] Number of active timers: ", len(activeTimers))

	// Check if there's an active timer
	if len(activeTimers) == 0 {
		logger.Info("[api][ApiGetTimer] No active timer found")
		return c.JSON(fiber.Map{
			"is_active": false,
			"message":   "No active timer",
		})
	}

	// Get the first (and only) active timer
	timerID := activeTimers[0]
	activeTimer, exists := timer.GetTimer(timerID)
	if !exists || !activeTimer.IsActive() {
		logger.Warning("[api][ApiGetTimer] Timer '" + timerID + "' is not active or not found")
		return c.JSON(fiber.Map{
			"is_active": false,
			"message":   "No active timer",
		})
	}

	// Calculate remaining time
	expireTime := activeTimer.GetExpireTime()
	timeRemaining := time.Until(expireTime)

	logger.Info("[api][ApiGetTimer] Active timer found: " + timerID)
	logger.Debug("[api][ApiGetTimer] Expire time: " + expireTime.Format(time.RFC3339))
	logger.Debug("[api][ApiGetTimer] Time remaining: " + timeRemaining.String())

	// Format response
	return c.JSON(fiber.Map{
		"is_active":      true,
		"timer_id":       timerID,
		"expire_time":    expireTime.Format(time.RFC3339),
		"current_time":   time.Now().Format(time.RFC3339),
		"time_remaining": timeRemaining.String(),
		"seconds_left":   int64(timeRemaining.Seconds()),
		"minutes_left":   int64(timeRemaining.Minutes()),
		"message":        "Active timer found",
	})
}
