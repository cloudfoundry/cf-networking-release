package planner

import "sync/atomic"

type LoggingState struct {
	isEnabled int32
}

func (l *LoggingState) IsEnabled() bool {
	return atomic.LoadInt32(&l.isEnabled) == 1
}

func (l *LoggingState) Enable() {
	atomic.StoreInt32(&l.isEnabled, 1)
	return
}

func (l *LoggingState) Disable() {
	atomic.StoreInt32(&l.isEnabled, 0)
	return
}
