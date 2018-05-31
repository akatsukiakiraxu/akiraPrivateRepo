package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	flag.Usage = func() { flag.PrintDefaults() }
	discoveryAddress := flag.String("address", "127.0.0.1", "Address of olive unit.")
	discoveryPort := flag.Uint("port", 6000, "Port of olive unit.")
	flag.Parse()

	ticker := time.NewTicker(time.Millisecond * 500)
	const TransferCount int = 8
	const NumberOfChannels int = 1
	const BytesPerSample int = 2
	const BufferSize int = NumberOfChannels * BytesPerSample * 1000000
	const HeaderSize = 16
	const Signature = uint32(0x74733181) // ts1?

	var transmitBuffer [BufferSize + HeaderSize + 4]byte
	headerBuffer := transmitBuffer[:HeaderSize]
	sequenceNumberBuffer := transmitBuffer[HeaderSize : HeaderSize+4]
	dataBuffer := transmitBuffer[HeaderSize+4:]
	var transferIndex int
	var counter uint16
	var sequenceNumber uint32

	var requestBuffer [1024]byte

	localAddr, err := net.ResolveTCPAddr("tcp4", "127.0.0.1:0")
	if err != nil {
		log.Printf("Failed to resolve local address. %v", err)
		return
	}
	remoteAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", *discoveryAddress, *discoveryPort))
	if err != nil {
		log.Printf("Failed to resolve remote address. %v", err)
		return
	}

	var conn *net.TCPConn
	for {
		select {
		case <-ticker.C:
			var err error
			if conn == nil {
				conn, err = net.DialTCP("tcp", localAddr, remoteAddr)
				if err != nil {
					continue
				}
			}

			var flags uint32
			if transferIndex == 0 {
				flags = 1
			} else if transferIndex == TransferCount-1 {
				flags = 2
			}

			binary.LittleEndian.PutUint32(headerBuffer[0:4], Signature)
			binary.LittleEndian.PutUint32(headerBuffer[4:8], 0)
			binary.LittleEndian.PutUint32(headerBuffer[8:12], flags)
			binary.LittleEndian.PutUint32(headerBuffer[12:16], uint32(BufferSize+4))
			binary.LittleEndian.PutUint32(sequenceNumberBuffer, sequenceNumber)
			sequenceNumber++
			for i := 0; i < BufferSize/NumberOfChannels; i += BytesPerSample {
				sampleValue := (counter + uint16(i>>4)) & 0x7fff
				for c := 0; c < NumberOfChannels; c++ {
					binary.LittleEndian.PutUint16(dataBuffer[c*BufferSize/NumberOfChannels+i:], sampleValue)
				}
			}

			conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
			var bytesWritten int
			for bytesWritten < len(transmitBuffer) {
				if n, err := conn.Write(transmitBuffer[bytesWritten:]); err != nil {
					log.Printf("Failed to transmit. %v\n", err)
					conn = nil
					break
				} else {
					bytesWritten += n
				}
			}
			log.Printf("Seq: %d\n", sequenceNumber)

			if transferIndex == TransferCount-1 {
				transferIndex = 0
			} else {
				transferIndex++
			}
			counter += uint16((BufferSize >> 4) & 0xffff)
			counter = counter & 0x7fff

			conn.SetReadDeadline(time.Now().Add(time.Millisecond * 1000))
			if n, err := conn.Read(requestBuffer[:]); err == nil {
				log.Printf("Received, %d", n)
				if n >= 0x18 {
					signature := binary.LittleEndian.Uint32(requestBuffer[0x00:0x04])
					packetType := binary.LittleEndian.Uint32(requestBuffer[0x04:0x08])
					length := binary.LittleEndian.Uint32(requestBuffer[0x0c:0x10])
					log.Printf("Received, signature=%x, packetType=%x, length=%d", signature, packetType, length)
					if signature == Signature && packetType == 2 && length == 8 {
						channelIndex := binary.LittleEndian.Uint32(requestBuffer[0x10:0x14])
						gain := binary.LittleEndian.Uint16(requestBuffer[0x14:0x16])
						threshold := binary.LittleEndian.Uint16(requestBuffer[0x16:0x18])
						log.Printf("Command received, channelIndex=%d, gain=%d, threshold=%d", channelIndex, gain, threshold)

						// Send response
						binary.LittleEndian.PutUint32(requestBuffer[0x04:0x08], 1) // Type = response
						binary.LittleEndian.PutUint32(requestBuffer[0x0c:0x10], 8) // Length = 8
						binary.LittleEndian.PutUint32(requestBuffer[0x10:0x14], 2) // TargetType = 2
						binary.LittleEndian.PutUint32(requestBuffer[0x14:0x18], 0) // Code = 0
						conn.Write(requestBuffer[:0x18])
					}
				}
			} else {
				if netError, ok := err.(net.Error); ok {
					if !netError.Timeout() {
						conn.Close()
						conn = nil
					}
				}
			}
		}
	}
}
