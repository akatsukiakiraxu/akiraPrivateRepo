package recording

import (
	"log"
	acq "olived/acquisition"
	"sync"

	"github.com/google/uuid"
)

type recordingController struct {
	filterLock        sync.RWMutex
	filters           []RecordingFilter
	pendingFramesLock sync.RWMutex
	pendingFrames     map[uuid.UUID]pendingFrame
}

type pendingFrame struct {
	FrameData acq.FrameData
	Recorder  Recorder
}

func NewRecordingController() RecordingController {
	return &recordingController{
		filters:       make([]RecordingFilter, 0),
		pendingFrames: make(map[uuid.UUID]pendingFrame),
	}
}

func (r *recordingController) AddFilter(filter RecordingFilter) {
	r.filterLock.Lock()
	defer r.filterLock.Unlock()
	r.filters = append(r.filters, filter)
}

func (r *recordingController) FrameRecorded(recorder Recorder, frame *acq.FrameData) FilterResult {
	result := r.frameRecorded(recorder, frame, nil)
	log.Printf("FrameRecorded result=%s", result)
	return result
}

func (r *recordingController) frameRecorded(recorder Recorder, frame *acq.FrameData, initialFilter RecordingFilter) FilterResult {
	r.filterLock.RLock()
	defer r.filterLock.RUnlock()

	for _, filter := range r.filters {
		if initialFilter != nil {
			if initialFilter == filter {
				initialFilter = nil
			}
			continue
		}
		result := filter.FrameRecorded(r, frame)
		switch result {
		case FilterResult_Approved:
			return result
		case FilterResult_Rejected:
			return result
		case FilterResult_Pending:
			r.pendingFramesLock.Lock()
			defer r.pendingFramesLock.Unlock()
			r.pendingFrames[frame.FrameId()] = pendingFrame{
				FrameData: *frame,
				Recorder:  recorder,
			}
			return result
		}
	}
	return FilterResult_Rejected
}

func (r *recordingController) popPendingFrame(frameData *acq.FrameData) pendingFrame {
	r.pendingFramesLock.Lock()
	defer r.pendingFramesLock.Unlock()
	frameID := frameData.FrameId()
	result := r.pendingFrames[frameID]
	delete(r.pendingFrames, frameID)
	return result
}

func (r *recordingController) FrameConfirmed(filter RecordingFilter, frameData *acq.FrameData, result FilterResult) {
	pendingFrame := r.popPendingFrame(frameData)
	if result == FilterResult_NotInterested {
		// Not interested. Re-check filter list.
		result = r.frameRecorded(pendingFrame.Recorder, &pendingFrame.FrameData, filter)
	}
	pendingFrame.Recorder.PendingFrameConfirmed(pendingFrame.FrameData.FrameId(), result)
}
