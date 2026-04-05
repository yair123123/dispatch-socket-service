package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"dispatch-socket-service/internal/models"
	"dispatch-socket-service/internal/services"
)

type MessageRouter struct {
	presence *services.PresenceService
	location *services.LocationService
	accept   *services.RideAcceptanceService
	logger   *slog.Logger
}

func NewMessageRouter(p *services.PresenceService, l *services.LocationService, a *services.RideAcceptanceService, logger *slog.Logger) *MessageRouter {
	return &MessageRouter{presence: p, location: l, accept: a, logger: logger}
}

func (r *MessageRouter) Route(ctx context.Context, driverID string, raw []byte) error {
	var in struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return fmt.Errorf("invalid message envelope: %w", err)
	}
	switch in.Type {
	case "set_availability":
		var req models.SetAvailabilityRequest
		if err := json.Unmarshal(in.Payload, &req); err != nil {
			return err
		}
		return r.presence.SetAvailability(ctx, driverID, req.IsAvailable)
	case "location_update":
		var req models.LocationUpdateRequest
		if err := json.Unmarshal(in.Payload, &req); err != nil {
			return err
		}
		return r.location.UpdateLocation(ctx, driverID, req)
	case "accept_ride":
		var req models.AcceptRideRequest
		if err := json.Unmarshal(in.Payload, &req); err != nil {
			return err
		}
		_, err := r.accept.AcceptRide(ctx, driverID, req)
		return err
	case "reject_ride":
		var req models.RejectRideRequest
		if err := json.Unmarshal(in.Payload, &req); err != nil {
			return err
		}
		return r.accept.RejectRide(ctx, driverID, req)
	default:
		r.logger.Warn("unknown ws message", "driver_id", driverID, "type", in.Type)
		return nil
	}
}
