package ws

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"dispatch-socket-service/internal/auth"
	"dispatch-socket-service/internal/models"
	"dispatch-socket-service/internal/services"

	"github.com/gorilla/websocket"
)

type DriverWSHandler struct {
	auth         *auth.JWTAuthenticator
	connections  *ConnectionManager
	presence     *services.PresenceService
	router       *MessageRouter
	pingInterval time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	logger       *slog.Logger
	upgrader     websocket.Upgrader
}

func NewDriverWSHandler(a *auth.JWTAuthenticator, cm *ConnectionManager, p *services.PresenceService, router *MessageRouter, pingInterval, readTimeout, writeTimeout time.Duration, logger *slog.Logger) *DriverWSHandler {
	return &DriverWSHandler{
		auth: a, connections: cm, presence: p, router: router,
		pingInterval: pingInterval, readTimeout: readTimeout, writeTimeout: writeTimeout, logger: logger,
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (h *DriverWSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	driverID, err := h.auth.ExtractDriverID(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	if old := h.connections.Set(driverID, conn); old != nil {
		_ = old.Close()
	}
	_ = h.presence.SetOnline(r.Context(), driverID, true)
	_ = h.connections.SendToDriver(driverID, models.Envelope{Type: "connected", Payload: models.ConnectedPayload{DriverID: driverID}}, h.writeTimeout)

	conn.SetReadLimit(4096)
	_ = conn.SetReadDeadline(time.Now().Add(h.readTimeout))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(h.readTimeout))
		_ = h.presence.TouchLastSeen(context.Background(), driverID)
		return nil
	})

	done := make(chan struct{})
	go h.startHeartbeat(conn, done)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		_ = h.presence.TouchLastSeen(context.Background(), driverID)
		if err := h.router.Route(r.Context(), driverID, msg); err != nil {
			h.logger.Warn("route ws message failed", "driver_id", driverID, "error", err)
		}
	}
	close(done)
	h.connections.Remove(driverID, conn)
	_ = h.presence.SetOnline(context.Background(), driverID, false)
	_ = conn.Close()
}

func (h *DriverWSHandler) startHeartbeat(conn *websocket.Conn, done <-chan struct{}) {
	ticker := time.NewTicker(h.pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(h.writeTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
				return
			}
		}
	}
}
