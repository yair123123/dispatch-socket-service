package rediskeys

type DriverState struct {
	DriverID      string `json:"driver_id"`
	IsOnline      bool   `json:"is_online"`
	IsAvailable   bool   `json:"is_available"`
	Lat           string `json:"lat"`
	Lon           string `json:"lon"`
	Accuracy      string `json:"accuracy"`
	Speed         string `json:"speed"`
	Heading       string `json:"heading"`
	LastSeenAt    string `json:"last_seen_at"`
	CurrentRideID string `json:"current_ride_id,omitempty"`
}
