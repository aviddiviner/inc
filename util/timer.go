package util

import "time"

type IntervalTimer interface {
	Stop()
	// TODO: Lap()
	// TODO: AvgLapTime() float
}

type intervalTimer struct {
	ticker *time.Ticker
	quit   chan struct{}
}

// Start a timer that will call back to a function every N milliseconds.
func NewTimer(millis int, f func()) IntervalTimer {
	t := &intervalTimer{
		ticker: time.NewTicker(time.Duration(millis) * time.Millisecond),
		quit:   make(chan struct{}),
	}
	go func() {
		for {
			select {
			case <-t.ticker.C:
				f()
			case <-t.quit:
				t.ticker.Stop()
				return
			}
		}
	}()
	return t
}

func (t *intervalTimer) Stop() {
	close(t.quit)
}
