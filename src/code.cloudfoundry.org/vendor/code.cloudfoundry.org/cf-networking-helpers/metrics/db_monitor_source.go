package metrics

import "code.cloudfoundry.org/bbs/db/sqldb/helpers/monitor"

type Db interface {
	OpenConnections() int
}

func NewDBMonitorSource(Db Db, monitor monitor.Monitor) []MetricSource {
	return []MetricSource{
		{
			Name: "DBOpenConnections",
			Unit: "",
			Getter: func() (float64, error) {
				return float64(Db.OpenConnections()), nil
			},
		},
		{
			Name: "DBQueriesTotal",
			Unit: "",
			Getter: func() (float64, error) {
				return float64(monitor.Total()), nil
			},
		},
		{
			Name: "DBQueriesSucceeded",
			Unit: "",
			Getter: func() (float64, error) {
				return float64(monitor.Succeeded()), nil
			},
		},
		{
			Name: "DBQueriesFailed",
			Unit: "",
			Getter: func() (float64, error) {
				return float64(monitor.Failed()), nil
			},
		},
		{
			Name: "DBQueriesInFlight",
			Unit: "",
			Getter: func() (float64, error) {
				return float64(monitor.ReadAndResetInFlightMax()), nil
			},
		},
		{
			Name: "DBQueryDurationMax",
			Unit: "seconds",
			Getter: func() (float64, error) {
				return monitor.ReadAndResetDurationMax().Seconds(), nil
			},
		},
	}
}
