package mywebsocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Session represents a single user WebSocket session that is controlling a device.
type Session struct {
	UserID   string
	DeviceID string
	Conn     *websocket.Conn
}

// SessionManager manages all active device + user sessions in memory.
type SessionManager struct {
	mu sync.RWMutex

	// deviceId -> device websocket connection
	devices map[string]*websocket.Conn

	// userId -> user session
	users map[string]*Session

	// deviceId -> userId (who controls this device)
	userByDevice map[string]string
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		devices:      make(map[string]*websocket.Conn),
		users:        make(map[string]*Session),
		userByDevice: make(map[string]string),
	}
}

// ========== Devices ==========

func (sm *SessionManager) AddDevice(deviceID string, conn *websocket.Conn) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.devices[deviceID] = conn
}

func (sm *SessionManager) RemoveDevice(deviceID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.devices, deviceID)
	delete(sm.userByDevice, deviceID)
	// NOTE: we do NOT delete userByDevice here,
	// because user might reconnect their device later.
	// If you want strict cleanup, you can also delete(sm.userByDevice, deviceID).
}

func (sm *SessionManager) GetDeviceConn(deviceID string) *websocket.Conn {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.devices[deviceID]
}

// ========== Users ==========

func (sm *SessionManager) AddUser(userID, deviceID string, conn *websocket.Conn) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.users[userID] = &Session{
		UserID:   userID,
		DeviceID: deviceID,
		Conn:     conn,
	}

	// 1 device -> 1 controlling user
	sm.userByDevice[deviceID] = userID
}

func (sm *SessionManager) RemoveUser(userID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if s, ok := sm.users[userID]; ok {
		// Remove mapping device -> user
		delete(sm.userByDevice, s.DeviceID)
	}
	delete(sm.users, userID)
}

func (sm *SessionManager) GetUser(userID string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.users[userID]
}

// ========== One-user-per-device lookup ==========

// GetUserByDevice returns the single user session controlling this device,
// or nil if none.
func (sm *SessionManager) GetUserByDevice(deviceID string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	userID, ok := sm.userByDevice[deviceID]
	if !ok {
		return nil
	}
	return sm.users[userID]
}
