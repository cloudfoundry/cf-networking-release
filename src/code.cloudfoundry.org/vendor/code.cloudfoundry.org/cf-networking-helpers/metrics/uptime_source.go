package metrics

import "time"

func NewUptimeSource() MetricSource {
	startTime := time.Now().Unix()
	return MetricSource{
		Name: "uptime",
		Unit: "seconds",
		Getter: func() (float64, error) {
			uptime := time.Now().Unix() - startTime
			return float64(uptime), nil
		},
	}
}
