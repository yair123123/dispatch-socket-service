package rediskeys

import "fmt"

const DriversLocationsKey = "drivers:locations"
const CoreSyncRetryQueueKey = "core:sync:retry:queue"

func DriverStateKey(driverID string) string { return fmt.Sprintf("driver:state:%s", driverID) }
func RideDispatchKey(rideID string) string  { return fmt.Sprintf("ride:%s:dispatch", rideID) }
func RideOfferKey(rideID, driverID string) string {
	return fmt.Sprintf("ride:%s:offer:%s", rideID, driverID)
}
func RideWinnerKey(rideID string) string { return fmt.Sprintf("ride:%s:winner", rideID) }
func RideRoundDriversKey(rideID string, roundNumber int) string {
	return fmt.Sprintf("ride:%s:round:%d:drivers", rideID, roundNumber)
}
