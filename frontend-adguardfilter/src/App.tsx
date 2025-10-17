import { useState, useEffect } from 'react'

interface BlockedService {
  id: string
  name: string
  icon_svg: string | Uint8Array | null | undefined
  rules?: string[]
  group_id?: string
}

interface BlockedServicesResponse {
  ids: string[]
  schedule?: {
    time_zone?: string
  }
}

interface TimerResponse {
  is_active: boolean
  timer_id?: string
  expire_time?: string
  current_time?: string
  time_remaining?: string
  seconds_left?: number
  minutes_left?: number
  message: string
}

function App() {
  const [blockedServices, setBlockedServices] = useState<BlockedService[]>([])
  const [enabledServices, setEnabledServices] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [resetMinutes, setResetMinutes] = useState<number>(0)
  const [resetDateTime, setResetDateTime] = useState<string>('')
  const [resetMode, setResetMode] = useState<'minutes' | 'datetime' | null>(null)
  const [saveLoading, setSaveLoading] = useState(false)
  const [saveMessage, setSaveMessage] = useState<string | null>(null)
  const [activeTimer, setActiveTimer] = useState<TimerResponse | null>(null)
  const [timerCountdown, setTimerCountdown] = useState<number>(0)

  // Fetch blocked services from API
  useEffect(() => {
    const fetchBlockedServices = async () => {
      try {
        setLoading(true)

        // Fetch the list of all services
        console.log('Fetching services from http://localhost:3000/api/v1/getservicelist')
        const servicesResponse = await fetch('http://localhost:3000/api/v1/getservicelist', {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
        })
        console.log('Services response status:', servicesResponse.status)
        if (!servicesResponse.ok) {
          throw new Error(`Failed to fetch services list: ${servicesResponse.statusText}`)
        }
        const servicesData: BlockedService[] = await servicesResponse.json()
        console.log('Received services data:', servicesData)

        // Validate data structure
        if (!Array.isArray(servicesData)) {
          throw new Error('Services API response is not an array')
        }
        setBlockedServices(servicesData)

        // Fetch the currently blocked services
        console.log('Fetching blocked services from http://localhost:3000/api/v1/getblockedservices')
        const blockedResponse = await fetch('http://localhost:3000/api/v1/getblockedservices', {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
        })
        console.log('Blocked services response status:', blockedResponse.status)
        if (!blockedResponse.ok) {
          throw new Error(`Failed to fetch blocked services: ${blockedResponse.statusText}`)
        }
        const blockedData: BlockedServicesResponse = await blockedResponse.json()
        console.log('Received blocked services data:', blockedData)
        console.log('Blocked service IDs:', blockedData.ids)

        // Initialize enabled services from the response
        if (blockedData.ids && Array.isArray(blockedData.ids)) {
          setEnabledServices(new Set(blockedData.ids))
        } else {
          console.warn('No ids found in blocked services response, initializing as empty')
          setEnabledServices(new Set())
        }

        setError(null)
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Unknown error occurred'
        setError(errorMsg)
        console.error('Error fetching data:', err)
        setBlockedServices([])
        setEnabledServices(new Set())
      } finally {
        setLoading(false)
      }
    }

    fetchBlockedServices()
  }, [])

  // Fetch and monitor active timer
  useEffect(() => {
    const fetchTimer = async () => {
      try {
        const response = await fetch('http://localhost:3000/api/v1/gettimer', {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
        })

        if (!response.ok) {
          throw new Error(`Failed to fetch timer: ${response.statusText}`)
        }

        const timerData: TimerResponse = await response.json()
        console.log('Timer data:', timerData)

        if (timerData.is_active && timerData.seconds_left !== undefined) {
          setActiveTimer(timerData)
          setTimerCountdown(timerData.seconds_left)
        } else {
          setActiveTimer(null)
          setTimerCountdown(0)
        }
      } catch (err) {
        console.error('Error fetching timer:', err)
        setActiveTimer(null)
        setTimerCountdown(0)
      }
    }

    // Fetch timer immediately on mount
    fetchTimer()

    // Set up interval to update timer every second
    const timerInterval = setInterval(() => {
      setTimerCountdown(prev => {
        if (prev <= 1) {
          // Timer expired, fetch again to confirm
          fetchTimer()
          return 0
        }
        return prev - 1
      })
    }, 1000)

    // Set up interval to refresh timer from API every 5 seconds
    const refreshInterval = setInterval(() => {
      fetchTimer()
    }, 5000)

    return () => {
      clearInterval(timerInterval)
      clearInterval(refreshInterval)
    }
  }, [])

  // Handle toggle for a service
  const handleToggleService = async (serviceID: string) => {
    try {
      setEnabledServices(prev => {
        const newSet = new Set(prev)
        if (newSet.has(serviceID)) {
          newSet.delete(serviceID)
        } else {
          newSet.add(serviceID)
        }
        return newSet
      })

      // TODO: Send the updated state to the API
      // For now, just log it
      console.log(`Toggled service: ${serviceID}`)
    } catch (err) {
      console.error('Error toggling service:', err)
    }
  }

  // Handle save configuration
  const handleSaveConfiguration = async () => {
    try {
      setSaveLoading(true)
      setSaveMessage(null)

      // Determine which mode is active
      if (!resetMode) {
        setSaveMessage('❌ Please select either minutes or date/time option')
        setSaveLoading(false)
        return
      }

      // Convert Set to array
      const enabledIds = Array.from(enabledServices)

      let endpoint = ''
      let payload: unknown

      if (resetMode === 'minutes') {
        if (resetMinutes <= 0) {
          setSaveMessage('❌ Please enter a valid number of minutes')
          setSaveLoading(false)
          return
        }

        endpoint = 'http://localhost:3000/api/v1/updateblockedservicesmin'
        payload = {
          config: {
            schedule: {
              time_zone: 'America/Chicago',
            },
            ids: enabledIds,
          },
          reset_after_min: resetMinutes,
        }
      } else if (resetMode === 'datetime') {
        if (!resetDateTime) {
          setSaveMessage('❌ Please select a date and time')
          setSaveLoading(false)
          return
        }

        // Format the datetime to ISO string format (YYYY-MM-DDTHH:mm:ss)
        const datetime = new Date(resetDateTime)
        if (isNaN(datetime.getTime())) {
          setSaveMessage('❌ Invalid date/time format')
          setSaveLoading(false)
          return
        }

        const formattedDateTime = datetime.toISOString().slice(0, 19)

        endpoint = 'http://localhost:3000/api/v1/updateblockedservicesdatetime'
        payload = {
          config: {
            schedule: {
              time_zone: 'America/Chicago',
            },
            ids: enabledIds,
          },
          reset_date_time: formattedDateTime,
        }
      }

      console.log('Saving configuration:', payload)

      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      })

      console.log('Save response status:', response.status)

      if (!response.ok) {
        throw new Error(`Failed to save configuration: ${response.statusText}`)
      }

      setSaveMessage('✅ Configuration saved successfully!')
      console.log('Configuration saved successfully')

      // Clear message after 3 seconds
      setTimeout(() => {
        setSaveMessage(null)
      }, 3000)
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Unknown error occurred'
      setSaveMessage(`❌ ${errorMsg}`)
      console.error('Error saving configuration:', err)
    } finally {
      setSaveLoading(false)
    }
  }

  // Format remaining time in a human-readable way
  const formatTimeRemaining = (seconds: number): string => {
    if (seconds <= 0) return 'Expired'

    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60

    if (hours > 0) {
      return `${hours}h ${minutes}m ${secs}s`
    } else if (minutes > 0) {
      return `${minutes}m ${secs}s`
    } else {
      return `${secs}s`
    }
  }

  // Convert IconSVG byte array to data URL
  const iconToDataUrl = (icon_svg: string | Uint8Array | undefined | null): string => {
    if (!icon_svg) {
      // Return a placeholder data URL if no icon
      return `data:image/svg+xml;base64,${btoa('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="#ccc"/></svg>')}`
    }

    try {
      let svgString: string

      // If it's a string, it's either an SVG string or base64-encoded string
      if (typeof icon_svg === 'string') {
        // Check if it looks like SVG XML
        if (icon_svg.startsWith('<svg')) {
          svgString = icon_svg
        } else {
          // If not, assume it's base64 and try to decode it
          try {
            svgString = atob(icon_svg)
          } catch {
            // If decoding fails, treat it as raw SVG
            svgString = icon_svg
          }
        }
        return `data:image/svg+xml;base64,${btoa(svgString)}`
      }

      // If it's a byte array (array of numbers from JSON), convert it
      if (Array.isArray(icon_svg) || icon_svg instanceof Uint8Array) {
        const decoder = new TextDecoder()
        const bytes = icon_svg instanceof Uint8Array ? icon_svg : new Uint8Array(icon_svg as number[])
        svgString = decoder.decode(bytes)
        return `data:image/svg+xml;base64,${btoa(svgString)}`
      }

      // Fallback: use placeholder
      console.warn('Unknown icon_svg format:', typeof icon_svg)
      return `data:image/svg+xml;base64,${btoa('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="#ccc"/></svg>')}`
    } catch (e) {
      console.error('Error converting icon:', e, icon_svg)
      // Return placeholder if conversion fails
      return `data:image/svg+xml;base64,${btoa('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="#ccc"/></svg>')}`
    }
  }

  // Filter services based on search term
  const filteredServices = blockedServices.filter(service => {
    const name = service.name || ''
    const id = service.id || ''
    return name.toLowerCase().includes(searchTerm.toLowerCase()) ||
           id.toLowerCase().includes(searchTerm.toLowerCase())
  })

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-600 to-purple-700 flex items-center justify-center">
        <div className="bg-white rounded-lg shadow-lg p-8 text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-lg font-semibold text-gray-800">Loading blocked services...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-600 to-purple-700 flex items-center justify-center p-4">
        <div className="bg-white rounded-lg shadow-lg p-8 text-center max-w-md">
          <div className="text-red-600 text-4xl mb-4">⚠️</div>
          <p className="text-lg font-semibold text-gray-800 mb-2">Error Loading Services</p>
          <p className="text-gray-600 mb-4">{error}</p>
          <details className="text-left bg-gray-100 p-4 rounded mt-4">
            <summary className="cursor-pointer font-semibold">Debug Info</summary>
            <pre className="text-xs mt-2 overflow-auto max-h-64">
              {JSON.stringify(blockedServices, null, 2)}
            </pre>
          </details>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-600 to-purple-700 p-4 md:p-8">
      {/* Header */}
      <div className="max-w-7xl mx-auto mb-8">
        <div className="text-center mb-8">
          <h1 className="text-4xl md:text-5xl font-bold text-white mb-2">
            🛡️ AdGuard Filter
          </h1>
          <p className="text-lg text-blue-100">
            Manage and control blocked services
          </p>
        </div>

        {/* Active Timer Banner */}
        {timerCountdown > 0 && activeTimer?.is_active && (
          <div className="mb-6 p-4 rounded-lg bg-yellow-100 border-2 border-yellow-400 shadow-lg">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <span className="text-2xl">⏱️</span>
                <div>
                  <p className="font-semibold text-yellow-900">
                    Service Block Active
                  </p>
                  <p className="text-sm text-yellow-800">
                    Services will be reset in: <span className="font-bold text-lg">{formatTimeRemaining(timerCountdown)}</span>
                  </p>
                </div>
              </div>
              <div className="text-right">
                <p className="text-xs text-yellow-700">Expires at</p>
                <p className="text-sm font-semibold text-yellow-900">
                  {activeTimer?.expire_time ? new Date(activeTimer.expire_time).toLocaleTimeString() : ''}
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Search Bar */}
        <div className="mb-6">
          <input
            type="text"
            placeholder="Search services..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full px-4 py-3 rounded-lg border-2 border-white bg-white text-gray-800 placeholder-gray-500 focus:outline-none focus:border-yellow-300 transition-colors"
          />
        </div>

        {/* Reset Time Configuration */}
        <div className="mb-6 bg-white bg-opacity-10 rounded-lg p-6 backdrop-blur-sm border border-white border-opacity-20">
          <div className="flex flex-col md:flex-row items-center gap-4">
            <div className="flex-1">
              <label className="block text-white font-semibold mb-2">
                Block Duration (minutes)
              </label>
                            <input
                type="number"
                min="0"
                value={resetMinutes}
                onChange={(e) => {
                  const val = parseInt(e.target.value) || 0
                  setResetMinutes(val)
                  if (val > 0) {
                    setResetMode('minutes')
                    setResetDateTime('')
                  }
                }}
                placeholder="Enter number of minutes"
                className={`px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                  resetMode === 'datetime' ? 'opacity-50 bg-gray-100' : ''
                }`}
                disabled={resetMode === 'datetime'}
              />
            </div>
            <div className="flex-1">
              <label className="block text-white font-semibold mb-2">
                Block Until (Date & Time)
              </label>
              <input
                type="datetime-local"
                value={resetDateTime}
                onChange={(e) => {
                  const val = e.target.value
                  setResetDateTime(val)
                  if (val) {
                    setResetMode('datetime')
                    setResetMinutes(0)
                  }
                }}
                className={`px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                  resetMode === 'minutes' ? 'opacity-50 bg-gray-100' : ''
                }`}
                disabled={resetMode === 'minutes'}
              />
            </div>
            <button
              onClick={handleSaveConfiguration}
              disabled={saveLoading}
              className="md:mt-6 px-6 py-3 bg-green-500 hover:bg-green-600 disabled:bg-gray-400 text-white font-semibold rounded-lg transition-colors flex items-center gap-2"
            >
              {saveLoading ? (
                <>
                  <span className="inline-block animate-spin">⏳</span>
                  Saving...
                </>
              ) : (
                <>
                  💾 Save Configuration
                </>
              )}
            </button>
          </div>
          {saveMessage && (
            <div className={`mt-4 p-3 rounded-lg text-white font-medium ${
              saveMessage.startsWith('✅') ? 'bg-green-500' : 'bg-red-500'
            }`}>
              {saveMessage}
            </div>
          )}
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 gap-4 mb-6">
          <div className="bg-white bg-opacity-20 rounded-lg p-4 text-center backdrop-blur-sm">
            <p className="text-blue-100 text-sm font-medium">Total Services</p>
            <p className="text-white text-3xl font-bold">{blockedServices.length}</p>
          </div>
          <div className="bg-white bg-opacity-20 rounded-lg p-4 text-center backdrop-blur-sm">
            <p className="text-blue-100 text-sm font-medium">Enabled</p>
            <p className="text-green-200 text-3xl font-bold">{enabledServices.size}</p>
          </div>
          <div className="bg-white bg-opacity-20 rounded-lg p-4 text-center backdrop-blur-sm">
            <p className="text-blue-100 text-sm font-medium">Disabled</p>
            <p className="text-red-200 text-3xl font-bold">{blockedServices.length - enabledServices.size}</p>
          </div>
        </div>
      </div>

      {/* Services Grid */}
      <div className="max-w-7xl mx-auto">
        {filteredServices.length === 0 ? (
          <div className="bg-white rounded-lg shadow-lg p-8 text-center">
            <p className="text-gray-600 text-lg">
              {searchTerm ? 'No services found matching your search.' : 'No services available.'}
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {filteredServices.map((service, index) => (
              <div
                key={service.id || `service-${index}`}
                className="bg-white rounded-lg shadow-lg hover:shadow-xl hover:scale-105 transition-all p-4"
              >
                <div className="flex items-center justify-between">
                  {/* Left side: Icon and Info */}
                  <div className="flex items-center gap-3 flex-1">
                    <div className="flex-shrink-0">
                      <div className="w-12 h-12 bg-gray-100 rounded-lg flex items-center justify-center overflow-hidden">
                        <img
                          src={iconToDataUrl(service.icon_svg)}
                          alt={service.name}
                          className="w-full h-full object-contain p-1"
                          title={`Icon for ${service.name}`}
                          onError={(e) => {
                            // Fallback if SVG fails to load
                            console.error(`Failed to load icon for ${service.name}:`, e)
                            e.currentTarget.style.display = 'none'
                          }}
                          onLoad={() => {
                            console.log(`Icon loaded for ${service.name}`)
                          }}
                        />
                      </div>
                    </div>
                    <div className="flex-1 min-w-0">
                      <h3 className="text-lg font-semibold text-gray-800 truncate">
                        {service.name || 'Unknown Service'}
                      </h3>
                      <p className="text-sm text-gray-500 truncate">
                        {service.id || 'No ID'}
                      </p>
                    </div>
                  </div>

                  {/* Right side: Toggle Switch */}
                  <div className="flex-shrink-0 ml-4">
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={enabledServices.has(service.id)}
                        onChange={() => handleToggleService(service.id)}
                        className="sr-only peer"
                      />
                      <div className="w-11 h-6 bg-gray-300 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                    </label>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="max-w-7xl mx-auto mt-12 text-center">
        <p className="text-blue-100 text-sm">
          Showing {filteredServices.length} of {blockedServices.length} services
        </p>
      </div>
    </div>
  )
}

export default App
