package interval

import (
	"fmt"
	acq "olived/acquisition"
	"olived/recording"
	"sync"
	"time"
)

type IntervalFilterTimer interface {
	Now() time.Time
}

type DefaultIntervalFilterTimer struct{}

func (t *DefaultIntervalFilterTimer) Now() time.Time {
	return time.Now()
}

type intervalFilter struct {
	timer    IntervalFilterTimer
	lastTime *time.Time
	interval time.Duration
	lock     sync.Mutex
}

func NewIntervalFilter(timer IntervalFilterTimer) *intervalFilter {
	if timer == nil {
		timer = &DefaultIntervalFilterTimer{}
	}
	return &intervalFilter{
		timer: timer,
	}
}

func (f *intervalFilter) FrameRecorded(controller recording.RecordingController, frameData *acq.FrameData) recording.FilterResult {
	now := f.timer.Now()
	f.lock.Lock()
	defer f.lock.Unlock()
	result := recording.FilterResult_NotInterested
	if f.interval != 0 {
		if f.lastTime == nil || now.Sub(*f.lastTime) >= f.interval {
			result = recording.FilterResult_Approved
			f.lastTime = &now
		}
	}
	return result
}

func (f *intervalFilter) Interval() time.Duration { return f.interval }
func (f *intervalFilter) SetInterval(interval time.Duration) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.interval = interval
	f.lastTime = nil
}

func (f *intervalFilter) Reset() {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.lastTime = nil
}

func (f *intervalFilter) Configure(config map[string]interface{}) error {
	if value, ok := config["interval"]; ok {
		if interval, ok := value.(float64); ok {
			f.SetInterval(time.Duration(interval))
		} else {
			return fmt.Errorf("invalid type of interval parameter")
		}
	}
	return nil
}
