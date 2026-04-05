package models

type SendOfferRequest struct {
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
