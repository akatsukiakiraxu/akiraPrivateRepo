package interval

import (
	acq "olived/acquisition"
	"olived/recording"
	"time"

	"testing"
)

func checkFilterResult(t *testing.T, expected recording.FilterResult, actual recording.FilterResult, message string) {
	if expected != actual {
		t.Fatalf("%s expected=%s, actual=%s", message, expected, actual)
	}
}

func newFrameData() *acq.FrameData {
	f := acq.NewFrameData(make(map[string]interface{}))
	return &f
}

type testTimer struct {
	Time time.Time
}

func (t *testTimer) Now() time.Time { return t.Time }
func (t *testTimer) Set(time time.Time) {
	t.Time = time
}
func (t *testTimer) Advance(duration time.Duration) {
	t.Time = t.Time.Add(duration)
}

func TestZeroInterval(t *testing.T) {
	timer := &testTimer{}

	filter := NewIntervalFilter(timer)
	c := recording.NewRecordingController()
	f := newFrameData()
	result := filter.FrameRecorded(c, f)
	checkFilterResult(t, recording.FilterResult_NotInterested, result, "result mismatch")
}

func TestFirstTime(t *testing.T) {
	timer := &testTimer{}

	filter := NewIntervalFilter(timer)
	filter.SetInterval(time.Second)
	c := recording.NewRecordingController()
	f := newFrameData()
	result := filter.FrameRecorded(c, f)
	checkFilterResult(t, recording.FilterResult_Approved, result, "result mismatch")
}

func TestSecondTimeBeforeInterval(t *testing.T) {
	timer := &testTimer{}

	filter := NewIntervalFilter(timer)
	filter.SetInterval(time.Second)
	c := recording.NewRecordingController()
	f := newFrameData()
	filter.FrameRecorded(c, f)           // First time
	result := filter.FrameRecorded(c, f) // Second time before interval.
	checkFilterResult(t, recording.FilterResult_NotInterested, result, "result mismatch")
}

func TestSecondTimeAfterInterval(t *testing.T) {
	timer := &testTimer{}

	filter := NewIntervalFilter(timer)
	filter.SetInterval(time.Second)
	c := recording.NewRecordingController()
	f := newFrameData()
	filter.FrameRecorded(c, f)           // First time
	timer.Advance(time.Second)           // Advance timer
	result := filter.FrameRecorded(c, f) // Second time just after interval.
	checkFilterResult(t, recording.FilterResult_Approved, result, "result mismatch")
}

func TestAfterInterval(t *testing.T) {
	timer := &testTimer{}

	filter := NewIntervalFilter(timer)
	filter.SetInterval(time.Second)
	c := recording.NewRecordingController()
	f := newFrameData()
	filter.FrameRecorded(c, f) // First time
	filter.FrameRecorded(c, f) // Second time

	timer.Advance(time.Second)           // Advance timer
	result := filter.FrameRecorded(c, f) // Third time just after interval.
	checkFilterResult(t, recording.FilterResult_Approved, result, "3rd")
	result = filter.FrameRecorded(c, f) // 4th time
	checkFilterResult(t, recording.FilterResult_NotInterested, result, "4th")

	timer.Advance(time.Second / 2)      // Advance a half of interval
	result = filter.FrameRecorded(c, f) // 5th time
	checkFilterResult(t, recording.FilterResult_NotInterested, result, "5th")

	timer.Advance(time.Second / 2)      // Advance a half of interval
	result = filter.FrameRecorded(c, f) // 6th time
	checkFilterResult(t, recording.FilterResult_Approved, result, "6th")
}
