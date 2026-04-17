package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dispatch-socket-service/internal/clients"
	"dispatch-socket-service/internal/models"

	"github.com/stretchr/testify/require"
)

func TestCoreClientRoundResultPayloadAndSecret(t *testing.T) {
	var got models.DispatchRoundResultRequest
	secret := "s3cr3t"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/internal/dispatch/round-result", r.URL.Path)
		require.Equal(t, secret, r.Header.Get("X-Internal-Secret"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := clients.NewCoreClient(srv.URL, secret, 2*time.Second)
	winner := int64(55)
	err := c.ReportDispatchRoundResult(context.Background(), models.DispatchRoundResultRequest{RideID: 123, RoundID: "ride_123_round_1", RoundNumber: 1, Status: "winner_selected", WinnerDriverID: &winner})
	require.NoError(t, err)
	require.Equal(t, int64(123), got.RideID)
	require.Equal(t, "winner_selected", got.Status)
	require.NotNil(t, got.WinnerDriverID)
	require.Equal(t, int64(55), *got.WinnerDriverID)
}
