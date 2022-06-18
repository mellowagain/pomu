package uploader

import (
	"bytes"
	"fmt"
	"log"
	"pomu/s3"
)

type Uploader struct {
	segments  chan []byte
	buffer    []byte
	maxBuffer int
	id        string
	segmentID int
	s3        *s3.Client
}

func New(segments chan []byte, id string, s3 *s3.Client, maxBuffer int) *Uploader {
	return &Uploader{
		segments:  segments,
		buffer:    make([]byte, 0, maxBuffer),
		maxBuffer: maxBuffer,
		id:        id,
		segmentID: 0,
		s3:        s3,
	}
}

func (up *Uploader) flushBuffer() {
	segmentName := fmt.Sprintf("%s/%03d.ts", up.id, up.segmentID)
	up.s3.Upload(segmentName, bytes.NewReader(up.buffer))
	log.Println("Uploaded segment", segmentName)
	up.segmentID += 1
	up.buffer = []byte{}
}

func (up *Uploader) ProcessSegments() {
	for segment := range up.segments {
		up.buffer = append(up.buffer, segment...)
		if len(up.buffer) > up.maxBuffer {
			// Write this segment to S3
			up.flushBuffer()
		}
	}
	// No more segments, upload what we have left
	up.flushBuffer()
}
