package main

import (
	"encoding/binary"
	"math"
	"os"
	"time"
)

func main() {
	ticker := time.NewTicker(time.Millisecond * 10)
	const TransferCount int = 8
	const NumberOfChannels int = 4
	const BytesPerSample int = 8
	const BufferSize int = NumberOfChannels * BytesPerSample * 1024

	var headerBuffer [16]byte
	var dataBuffer [BufferSize * NumberOfChannels]byte
	var transferIndex int
	var counter uint16

	for {
		select {
		case <-ticker.C:
			var flags uint16
			if transferIndex == 0 {
				flags = 1
			} else if transferIndex == TransferCount-1 {
				flags = 2
			}

			binary.LittleEndian.PutUint16(headerBuffer[0:2], 3)
			binary.LittleEndian.PutUint16(headerBuffer[2:4], flags)
			binary.LittleEndian.PutUint32(headerBuffer[4:8], uint32(len(dataBuffer)+8))
			binary.LittleEndian.PutUint64(headerBuffer[8:16], uint64(0xf))

			for i := 0; i < BufferSize/NumberOfChannels; i += BytesPerSample {
				sampleValue := float64((counter+uint16(i)))/32768.0 - 1.0
				for c := 0; c < NumberOfChannels; c++ {
					value := math.Float64bits(float64(sampleValue))
					binary.LittleEndian.PutUint64(dataBuffer[c*BufferSize/NumberOfChannels+i:], value)
				}
			}

			os.Stdout.Write(headerBuffer[:])
			os.Stdout.Write(dataBuffer[:])

			if transferIndex == TransferCount-1 {
				transferIndex = 0
			} else {
				transferIndex++
			}
			counter += uint16(BufferSize >> 4)
			counter = counter & 0x7fff
		}

	}
}
