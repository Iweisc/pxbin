package logging

import "time"

// Timer helps measure request latency.
type Timer struct {
	start time.Time
}

func NewTimer() *Timer {
	return &Timer{start: time.Now()}
}

func (t *Timer) ElapsedMS() int {
	return int(time.Since(t.start).Milliseconds())
}
