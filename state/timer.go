package state

import (
	"time"

	"github.com/feichai0017/GoChat/common/timingwheel"
)

var wheel *timingwheel.TimingWheel

func InitTimer() {
	wheel = timingwheel.NewTimingWheel(time.Millisecond, 20)
	wheel.Start()
}
func CloseTimer() {
	wheel.Stop()
}

func AfterFunc(d time.Duration, f func()) *timingwheel.Timer {
	t := wheel.AfterFunc(d, f)
	return t
}