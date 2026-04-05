package services

import "time"

type DriverSender interface {
	SendToDriver(driverID string, message interface{}, writeTimeout time.Duration) error
	SendToDrivers(driverIDs []string, message interface{}, writeTimeout time.Duration) (sent []string, notConnected []string, failed []string)
}
