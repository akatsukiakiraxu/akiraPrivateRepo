package recording

import (
	acq "olived/acquisition"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

type testRecorder struct {
	RemovedFiles []uuid.UUID
}

func (r *testRecorder) AddMetadata(frameID uuid.UUID, key string, value interface{}) error {
	return nil
}
func (r *testRecorder) PendingFrameConfirmed(frameID uuid.UUID, result FilterResult) {
	if result == FilterResult_Rejected {
		r.RemovedFiles = append(r.RemovedFiles, frameID)
	}
}

func newTestRecorder() *testRecorder {
	return &testRecorder{
		RemovedFiles: make([]uuid.UUID, 0),
	}
}

var testApprovedFilter = &StaticRecordingFilter{Result: FilterResult_Approved}
var testRejectedFilter = &StaticRecordingFilter{Result: FilterResult_Rejected}
var testNotInterestedFilter = &StaticRecordingFilter{Result: FilterResult_NotInterested}
var testPendingFilter = &StaticRecordingFilter{Result: FilterResult_Pending}

func newFrameData() *acq.FrameData {
	f := acq.NewFrameData(make(map[string]interface{}))
	return &f
}

func (actual FilterResult) checkFilterResult(t *testing.T, expected FilterResult, message string) {
	if expected != actual {
		t.Fatalf("%s expected=%s, actual=%s", message, expected, actual)
	}
}

func testFilters(t *testing.T, filters []RecordingFilter, expected FilterResult) (*testRecorder, RecordingController, *acq.FrameData) {
	tr := newTestRecorder()
	c := NewRecordingController()
	f := newFrameData()
	for _, filter := range filters {
		c.AddFilter(filter)
	}
	result := c.FrameRecorded(tr, f)
	result.checkFilterResult(t, expected, "invalid filter")
	return tr, c, f
}

func TestEmptyController(t *testing.T) {
	filters := []RecordingFilter{}
	testFilters(t, filters, FilterResult_Rejected)
}

func TestApprovedFilter(t *testing.T) {
	filters := []RecordingFilter{testApprovedFilter}
	testFilters(t, filters, FilterResult_Approved)
}

func TestRejectedFilter(t *testing.T) {
	filters := []RecordingFilter{testRejectedFilter}
	testFilters(t, filters, FilterResult_Rejected)
}

func TestNotInterestedFilter(t *testing.T) {
	filters := []RecordingFilter{testNotInterestedFilter}
	testFilters(t, filters, FilterResult_Rejected)
}

func TestPendingFilter(t *testing.T) {
	filters := []RecordingFilter{testPendingFilter}
	testFilters(t, filters, FilterResult_Pending)
}

func TestFilters_AR(t *testing.T) {
	filters := []RecordingFilter{testApprovedFilter, testRejectedFilter}
	testFilters(t, filters, FilterResult_Approved)
}

func TestFilters_RA(t *testing.T) {
	filters := []RecordingFilter{testRejectedFilter, testApprovedFilter}
	testFilters(t, filters, FilterResult_Rejected)
}

func TestFilters_NA(t *testing.T) {
	filters := []RecordingFilter{testNotInterestedFilter, testApprovedFilter}
	testFilters(t, filters, FilterResult_Approved)
}

func TestFilters_NR(t *testing.T) {
	filters := []RecordingFilter{testNotInterestedFilter, testRejectedFilter, testApprovedFilter}
	testFilters(t, filters, FilterResult_Rejected)
}

func TestFilters_PA(t *testing.T) {
	filters := []RecordingFilter{testPendingFilter, testApprovedFilter}
	testFilters(t, filters, FilterResult_Pending)
}
func TestFilters_PR(t *testing.T) {
	filters := []RecordingFilter{testPendingFilter, testRejectedFilter}
	testFilters(t, filters, FilterResult_Pending)
}
func TestFilters_PN(t *testing.T) {
	filters := []RecordingFilter{testPendingFilter, testNotInterestedFilter}
	testFilters(t, filters, FilterResult_Pending)
}

func TestPendingFilter_Approve(t *testing.T) {
	filters := []RecordingFilter{testPendingFilter, testNotInterestedFilter}
	tr, c, f := testFilters(t, filters, FilterResult_Pending)
	c.FrameConfirmed(testPendingFilter, f, FilterResult_Approved)
	if len(tr.RemovedFiles) != 0 {
		t.Fatalf("length of RemovedFiles must be 0, actual = %d", len(tr.RemovedFiles))
	}
}

func TestPendingFilter_Reject(t *testing.T) {
	filters := []RecordingFilter{testPendingFilter, testNotInterestedFilter}
	tr, c, f := testFilters(t, filters, FilterResult_Pending)
	c.FrameConfirmed(testPendingFilter, f, FilterResult_Rejected)
	if len(tr.RemovedFiles) != 1 {
		t.Fatalf("length of RemovedFiles must be 1, actual = %d", len(tr.RemovedFiles))
	}
	if !reflect.DeepEqual(tr.RemovedFiles[0], f.FrameId()) {
		t.Fatalf("invalid removed file. expected=%v, actual=%v", f.FrameId(), tr.RemovedFiles[0])
	}
}

func TestPendingFilter_NotInterested(t *testing.T) {
	filters := []RecordingFilter{testPendingFilter, testNotInterestedFilter}
	tr, c, f := testFilters(t, filters, FilterResult_Pending)
	c.FrameConfirmed(testPendingFilter, f, FilterResult_NotInterested)
	if len(tr.RemovedFiles) != 1 {
		t.Fatalf("length of RemovedFiles must be 1, actual = %d", len(tr.RemovedFiles))
	}
	if !reflect.DeepEqual(tr.RemovedFiles[0], f.FrameId()) {
		t.Fatalf("invalid removed file. expected=%v, actual=%v", f.FrameId(), tr.RemovedFiles[0])
	}
}

func TestPendingFilter_DualPending(t *testing.T) {
	testPendingFilter2 := *testPendingFilter
	testNotInterestedFilter2 := *testNotInterestedFilter
	filters := []RecordingFilter{testPendingFilter, testNotInterestedFilter, &testPendingFilter2, &testNotInterestedFilter2}
	tr, c, f := testFilters(t, filters, FilterResult_Pending)
	c.FrameConfirmed(testPendingFilter, f, FilterResult_NotInterested)
	if len(tr.RemovedFiles) != 0 {
		t.Fatalf("length of RemovedFiles must be 0, actual = %d", len(tr.RemovedFiles))
	}
	c.FrameConfirmed(&testPendingFilter2, f, FilterResult_NotInterested)
	if len(tr.RemovedFiles) != 1 {
		t.Fatalf("length of RemovedFiles must be 1, actual = %d", len(tr.RemovedFiles))
	}
	if !reflect.DeepEqual(tr.RemovedFiles[0], f.FrameId()) {
		t.Fatalf("invalid removed file. expected=%v, actual=%v", f.FrameId(), tr.RemovedFiles[0])
	}
}
