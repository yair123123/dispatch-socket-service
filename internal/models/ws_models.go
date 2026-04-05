package models

type SetAvailabilityRequest struct {
	IsAvailable bool `json:"is_available"`
}

type LocationUpdateRequest struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Accuracy  float64 `json:"accuracy"`
	Speed     float64 `json:"speed"`
	Heading   float64 `json:"heading"`
	Timestamp string  `json:"timestamp"`
}

type AcceptRideRequest struct {
	RideID      string `json:"ride_id"`
	OfferID     string `json:"offer_id"`
	RoundNumber int    `json:"round_number"`
}

type RejectRideRequest struct {
	RideID      string `json:"ride_id"`
	OfferID     string `json:"offer_id"`
	RoundNumber int    `json:"round_number"`
}

type DriverWSMessage struct {
	Type    string `json:"type"`
	Payload []byte `json:"payload"`
}
