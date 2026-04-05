package models

type Envelope struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type ConnectedPayload struct {
	DriverID string `json:"driver_id"`
}

type RideOfferPayload struct {
	RideID      string       `json:"ride_id"`
	OfferID     string       `json:"offer_id"`
	RoundNumber int          `json:"round_number"`
	ExpiresAt   string       `json:"expires_at"`
	Price       float64      `json:"price"`
	Origin      LocationMeta `json:"origin"`
	Destination LocationMeta `json:"destination"`
}

type RideAssignedPayload struct {
	RideOfferPayload
	NextStep string `json:"next_step"`
}

type RideOfferCanceledPayload struct {
	RideID  string `json:"ride_id"`
	OfferID string `json:"offer_id"`
	Reason  string `json:"reason"`
}

type RideAcceptRejectedPayload struct {
	RideID  string `json:"ride_id"`
	OfferID string `json:"offer_id"`
	Reason  string `json:"reason"`
}

type LocationMeta struct {
	Label string  `json:"label"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
}
