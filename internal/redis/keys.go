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
func RoundResultSentKey(roundID string) string {
	return fmt.Sprintf("dispatch:round:%s:result_sent", roundID)
}
func H3CellDriversKey(cellID string) string {
	return fmt.Sprintf("h3:cell:%s:drivers", cellID)
}
func DriverH3CellKey(driverID string) string {
	return fmt.Sprintf("driver:%s:h3_cell", driverID)
}
