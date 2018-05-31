package recording

import (
	"fmt"
	acq "olived/acquisition"

	"github.com/google/uuid"
)

type FilterResult int

const (
	FilterResult_Approved FilterResult = iota
	FilterResult_Rejected
	FilterResult_Pending
	FilterResult_NotInterested
)

func (r FilterResult) String() string {
	switch r {
	case FilterResult_Approved:
		return "Approved"
	case FilterResult_Rejected:
		return "Rejected"
	case FilterResult_Pending:
		return "Pending"
	case FilterResult_NotInterested:
		return "NotInterested"
	default:
		return fmt.Sprintf("Unknown(%d)", int(r))
	}
}

type RecordingController interface {
	FrameRecorded(recorder Recorder, frameData *acq.FrameData) FilterResult
	FrameConfirmed(filter RecordingFilter, frameData *acq.FrameData, result FilterResult)

	AddFilter(filter RecordingFilter)
}

type RecordingFilter interface {
	FrameRecorded(controller RecordingController, frameData *acq.FrameData) FilterResult
}

type FrameRecordedFunc func(controller RecordingController, frameData *acq.FrameData) FilterResult
type FuncRecordingFilter struct {
	Func FrameRecordedFunc
}

func (f *FuncRecordingFilter) FrameRecorded(controller RecordingController, frameData *acq.FrameData) FilterResult {
	return f.Func(controller, frameData)
}

type StaticRecordingFilter struct {
	Result FilterResult
}

func (f *StaticRecordingFilter) FrameRecorded(controller RecordingController, frameData *acq.FrameData) FilterResult {
	return f.Result
}

type Recorder interface {
	AddMetadata(frameID uuid.UUID, key string, value interface{}) error
	PendingFrameConfirmed(frameID uuid.UUID, result FilterResult)
}
