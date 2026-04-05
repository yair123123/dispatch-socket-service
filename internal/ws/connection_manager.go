package ws

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ClientConn struct {
	Conn *websocket.Conn
	Mu   sync.Mutex
}

type ConnectionManager struct {
	mu    sync.RWMutex
	conns map[string]*ClientConn
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{conns: make(map[string]*ClientConn)}
}

func (m *ConnectionManager) Set(driverID string, conn *websocket.Conn) (old *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.conns[driverID]; ok {
		old = existing.Conn
	}
	m.conns[driverID] = &ClientConn{Conn: conn}
	return old
}

func (m *ConnectionManager) Remove(driverID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.conns[driverID]; ok && existing.Conn == conn {
		delete(m.conns, driverID)
	}
}

func (m *ConnectionManager) IsConnected(driverID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.conns[driverID]
	return ok
}

func (m *ConnectionManager) SendToDriver(driverID string, message interface{}, writeTimeout time.Duration) error {
	m.mu.RLock()
	client, ok := m.conns[driverID]
	m.mu.RUnlock()
	if !ok {
		return errors.New("driver not connected")
	}
	b, err := json.Marshal(message)
	if err != nil {
		return err
	}
	client.Mu.Lock()
	defer client.Mu.Unlock()
	if err := client.Conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return err
	}
	return client.Conn.WriteMessage(websocket.TextMessage, b)
}

func (m *ConnectionManager) SendToDrivers(driverIDs []string, message interface{}, writeTimeout time.Duration) (sent []string, notConnected []string, failed []string) {
	for _, driverID := range driverIDs {
		if !m.IsConnected(driverID) {
			notConnected = append(notConnected, driverID)
			continue
		}
		if err := m.SendToDriver(driverID, message, writeTimeout); err != nil {
			failed = append(failed, driverID)
			continue
		}
		sent = append(sent, driverID)
	}
	return
}
