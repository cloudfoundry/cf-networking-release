package metrics

import "time"

type NoOpMetricsSender struct{}

func (s *NoOpMetricsSender) SendDuration(string, time.Duration) {

}
func (s *NoOpMetricsSender) IncrementCounter(string) {
}
