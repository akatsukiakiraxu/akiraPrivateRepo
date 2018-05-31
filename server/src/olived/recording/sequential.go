package recording

import (
	"fmt"
	"io"
	"os"

	acq "olived/acquisition"
)

type sequentialRecorder struct {
	directory string

	started bool
	dataCh  chan acq.AcquiredData
	stopCh  chan struct{}
	doneCh  chan struct{}
}

func NewSequentialRecorder(directory string) *sequentialRecorder {
	return &sequentialRecorder{
		directory: directory,
		dataCh:    make(chan acq.AcquiredData, 2),
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

func (r *sequentialRecorder) Type() acq.ProcessorType {
	return acq.Recorder
}
func (r *sequentialRecorder) Started(unit acq.Unit) {
	go func() {
		var file io.ReadWriteCloser
		var counter uint32
		var err error
		defer func() {
			if file != nil {
				file.Close()
			}
			r.doneCh <- struct{}{}
		}()

		for {
			select {
			case <-r.stopCh:
				return
			case data := <-r.dataCh:
				if data.IsFrameFirstData() {
					if file != nil {
						file.Close()
					}
					file, err = os.Create(fmt.Sprintf("%s/%010d.bin", r.directory, counter))
					if err != nil {
						file = nil
					}
				}

				if file != nil {
					raw := []byte(data.RawData())
					bytesWritten := 0
					for bytesWritten < len(raw) {
						n, err := file.Write(raw[bytesWritten:])
						if err != nil {
							break
						}
						bytesWritten += n
					}
				}

				if data.IsFrameLastData() {
					if file != nil {
						file.Close()
					}
					file = nil
				}
			}
		}
	}()
	r.started = true
}
func (r *sequentialRecorder) Stopped(unit acq.Unit) {
	r.stopCh <- struct{}{}
	<-r.doneCh
	r.started = false
}

func (r *sequentialRecorder) SettingsChanging(unit acq.Unit, settings *acq.UnitSettings) error {
	return nil
}
func (r *sequentialRecorder) SettingsChanged(unit acq.Unit, settings *acq.UnitSettings) {
}
func (r *sequentialRecorder) DataArrived(unit acq.Unit, data acq.AcquiredData) {
	if r.started && data.Type() == acq.TimeSeries {
		r.dataCh <- data
	}
}
