package uploader

import "log"

type Uploader struct {
	segments  chan []byte
	buffer    []byte
	maxBuffer int
	writer    SegmentWriter
	segmentID int
}

func New(segments chan []byte, writer SegmentWriter, maxBuffer int) *Uploader {
	return &Uploader{
		segments:  segments,
		buffer:    make([]byte, 0, maxBuffer),
		maxBuffer: maxBuffer,
		segmentID: 0,
		writer:    writer,
	}
}

func (up *Uploader) ProcessSegments() {
	for segment := range up.segments {
		up.buffer = append(up.buffer, segment...)
		if len(up.buffer) > up.maxBuffer {
			// Write this segment to S3
			err := up.writer.write(up.segmentID, up.buffer)
			if err != nil {
				log.Println("Failed to write segment:", err)
			}
			up.segmentID += 1
			up.buffer = []byte{}
		}
	}
	// No more segments, upload what we have left
	up.writer.write(up.segmentID, up.buffer)
}
