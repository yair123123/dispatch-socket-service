package tests

import (
	"testing"
	"time"

	"dispatch-socket-service/internal/ws"

	"github.com/stretchr/testify/require"
)

func TestConnectionManagerAddGetRemove(t *testing.T) {
	cm := ws.NewConnectionManager()
	server, _, cleanup := makeWebsocketPair(t)
	defer cleanup()

	old := cm.Set("driver_1", server)
	require.Nil(t, old)
	require.True(t, cm.IsConnected("driver_1"))

	cm.Remove("driver_1", server)
	require.False(t, cm.IsConnected("driver_1"))
}

func TestConnectionManagerSend(t *testing.T) {
	cm := ws.NewConnectionManager()
	server, client, cleanup := makeWebsocketPair(t)
	defer cleanup()
	cm.Set("driver_1", server)

	err := cm.SendToDriver("driver_1", map[string]any{"type": "x"}, 2*time.Second)
	require.NoError(t, err)
	_, b, err := client.ReadMessage()
	require.NoError(t, err)
	require.Contains(t, string(b), "\"type\":\"x\"")
}
