package timer

import (
	"errors"
	"sync"
	"time"

	logger "github.com/welasco/adguardfilter/common/logger"
)

// Timer represents a timer that executes a callback function when it expires
type Timer struct {
	id          string
	timer       *time.Timer
	callback    func()
	expireTime  time.Time
	isActive    bool
	mu          sync.Mutex
	stopChan    chan bool
}

var (
	// Global timer registry to manage multiple timers
	timers   = make(map[string]*Timer)
	timersMu sync.RWMutex
)

// NewTimerWithDuration creates a new timer that expires after the specified duration in minutes
func NewTimerWithDuration(id string, minutes int, callback func()) (*Timer, error) {
	if minutes <= 0 {
		logger.Error("[timer][NewTimerWithDuration] Duration must be greater than 0")
		return nil, errors.New("duration must be greater than 0 minutes")
	}

	if callback == nil {
		logger.Error("[timer][NewTimerWithDuration] Callback function cannot be nil")
		return nil, errors.New("callback function cannot be nil")
	}

	duration := time.Duration(minutes) * time.Minute
	expireTime := time.Now().Add(duration)

	logger.Info("[timer][NewTimerWithDuration] Creating timer '" + id + "' for " + string(rune(minutes)) + " minutes")
	logger.Debug("[timer][NewTimerWithDuration] Timer will expire at: " + expireTime.Format(time.RFC3339))

	return createTimer(id, duration, expireTime, callback)
}

// NewTimerWithDeadline creates a new timer that expires at the specified date and time
func NewTimerWithDeadline(id string, deadline time.Time, callback func()) (*Timer, error) {
	if deadline.Before(time.Now()) {
		logger.Error("[timer][NewTimerWithDeadline] Deadline must be in the future")
		return nil, errors.New("deadline must be in the future")
	}

	if callback == nil {
		logger.Error("[timer][NewTimerWithDeadline] Callback function cannot be nil")
		return nil, errors.New("callback function cannot be nil")
	}

	duration := time.Until(deadline)

	logger.Info("[timer][NewTimerWithDeadline] Creating timer '" + id + "' with deadline: " + deadline.Format(time.RFC3339))
	logger.Debug("[timer][NewTimerWithDeadline] Duration until deadline: " + duration.String())

	return createTimer(id, duration, deadline, callback)
}

// createTimer is an internal function to create and start a timer
func createTimer(id string, duration time.Duration, expireTime time.Time, callback func()) (*Timer, error) {
	// Check if a timer with this ID already exists
	timersMu.Lock()
	if existingTimer, exists := timers[id]; exists {
		timersMu.Unlock()
		logger.Warning("[timer][createTimer] Timer '" + id + "' already exists. Stopping existing timer.")
		existingTimer.Stop()
		timersMu.Lock()
	}

	t := &Timer{
		id:         id,
		callback:   callback,
		expireTime: expireTime,
		isActive:   true,
		stopChan:   make(chan bool, 1),
	}

	// Register the timer
	timers[id] = t
	timersMu.Unlock()

	// Start the timer in a goroutine
	go t.run(duration)

	logger.Info("[timer][createTimer] Timer '" + id + "' started successfully")

	return t, nil
}

// run executes the timer and waits for expiration or cancellation
func (t *Timer) run(duration time.Duration) {
	t.timer = time.NewTimer(duration)

	select {
	case <-t.timer.C:
		// Timer expired - execute callback
		t.mu.Lock()
		t.isActive = false
		t.mu.Unlock()

		logger.Info("[timer][run] Timer '" + t.id + "' expired. Executing callback...")

		// Execute the callback function
		if t.callback != nil {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("[timer][run] Callback function panicked for timer '" + t.id + "'")
					logger.Error(r)
				}
			}()

			t.callback()
			logger.Info("[timer][run] Callback executed successfully for timer '" + t.id + "'")
		}

		// Remove from registry
		timersMu.Lock()
		delete(timers, t.id)
		timersMu.Unlock()

	case <-t.stopChan:
		// Timer was stopped manually
		t.mu.Lock()
		t.isActive = false
		t.mu.Unlock()

		if t.timer != nil {
			t.timer.Stop()
		}

		logger.Info("[timer][run] Timer '" + t.id + "' stopped manually")

		// Remove from registry
		timersMu.Lock()
		delete(timers, t.id)
		timersMu.Unlock()
	}
}

// Stop cancels the timer before it expires
func (t *Timer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isActive {
		logger.Debug("[timer][Stop] Timer '" + t.id + "' is already inactive")
		return
	}

	logger.Info("[timer][Stop] Stopping timer '" + t.id + "'")

	select {
	case t.stopChan <- true:
	default:
	}
}

// IsActive returns whether the timer is currently active
func (t *Timer) IsActive() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.isActive
}

// GetExpireTime returns the time when the timer will expire
func (t *Timer) GetExpireTime() time.Time {
	return t.expireTime
}

// GetID returns the timer's ID
func (t *Timer) GetID() string {
	return t.id
}

// GetTimer retrieves a timer by ID from the registry
func GetTimer(id string) (*Timer, bool) {
	timersMu.RLock()
	defer timersMu.RUnlock()
	timer, exists := timers[id]
	return timer, exists
}

// StopTimer stops a timer by ID
func StopTimer(id string) error {
	timersMu.RLock()
	timer, exists := timers[id]
	timersMu.RUnlock()

	if !exists {
		logger.Warning("[timer][StopTimer] Timer '" + id + "' not found")
		return errors.New("timer not found: " + id)
	}

	timer.Stop()
	return nil
}

// GetAllActiveTimers returns a list of all active timer IDs
func GetAllActiveTimers() []string {
	timersMu.RLock()
	defer timersMu.RUnlock()

	activeTimers := make([]string, 0, len(timers))
	for id, timer := range timers {
		if timer.IsActive() {
			activeTimers = append(activeTimers, id)
		}
	}

	return activeTimers
}

// StopAllTimers stops all active timers
func StopAllTimers() {
	timersMu.RLock()
	timerList := make([]*Timer, 0, len(timers))
	for _, timer := range timers {
		timerList = append(timerList, timer)
	}
	timersMu.RUnlock()

	logger.Info("[timer][StopAllTimers] Stopping all active timers")

	for _, timer := range timerList {
		timer.Stop()
	}
}
