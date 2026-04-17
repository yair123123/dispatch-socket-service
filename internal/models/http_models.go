package models

type SendOfferRequest struct {
	RoundID     string      `json:"round_id,omitempty"`
	RideID      string      `json:"ride_id" binding:"required"`
	RoundNumber int         `json:"round_number" binding:"required"`
	ExpiresAt   string      `json:"expires_at" binding:"required"`
	DriverIDs   []string    `json:"driver_ids" binding:"required"`
	Payload     OfferDetail `json:"payload" binding:"required"`
}

type OfferDetail struct {
	Price       float64      `json:"price"`
	Origin      LocationMeta `json:"origin"`
	Destination LocationMeta `json:"destination"`
}

type SendOfferResponse struct {
	Success               bool     `json:"success"`
	RideID                string   `json:"ride_id"`
	RoundNumber           int      `json:"round_number"`
	SentDriverIDs         []string `json:"sent_driver_ids"`
	NotConnectedDriverIDs []string `json:"not_connected_driver_ids"`
	FailedDriverIDs       []string `json:"failed_driver_ids"`
}

type CancelOfferRequest struct {
	RideID      string   `json:"ride_id" binding:"required"`
	DriverIDs   []string `json:"driver_ids" binding:"required"`
	OfferID     string   `json:"offer_id"`
	RoundNumber int      `json:"round_number"`
	Reason      string   `json:"reason" binding:"required"`
}

type CoreAssignDriverRequest struct {
	RideID      string `json:"ride_id"`
	DriverID    string `json:"driver_id"`
	OfferID     string `json:"offer_id"`
	RoundNumber int    `json:"round_number"`
}

type StartDispatchRoundRequest struct {
	RideID         int64           `json:"rideId" binding:"required"`
	RoundID        string          `json:"roundId" binding:"required"`
	RoundNumber    int             `json:"roundNumber" binding:"required"`
	StationID      int64           `json:"stationId"`
	OriginLat      float64         `json:"originLat" binding:"required"`
	OriginLon      float64         `json:"originLon" binding:"required"`
	RadiusKm       float64         `json:"radiusKm" binding:"required"`
	TimeoutSeconds int             `json:"timeoutSeconds" binding:"required,min=1"`
	MaxCandidates  int             `json:"maxCandidates" binding:"required,min=1"`
	RidePreview    DispatchPreview `json:"ridePreview" binding:"required"`
}

type DispatchPreview struct {
	OriginText      string  `json:"originText"`
	DestinationText string  `json:"destinationText"`
	Price           float64 `json:"price"`
	Note            string  `json:"note"`
}

type StartDispatchRoundResponse struct {
	Accepted bool   `json:"accepted"`
	RoundID  string `json:"roundId"`
}

type DispatchRoundResultRequest struct {
	RideID         int64  `json:"rideId"`
	RoundID        string `json:"roundId"`
	RoundNumber    int    `json:"roundNumber"`
	Status         string `json:"status"`
	WinnerDriverID *int64 `json:"winnerDriverId,omitempty"`
}
