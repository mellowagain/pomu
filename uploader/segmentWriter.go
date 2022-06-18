package uploader

import (
	"fmt"
	"log"
	"path"
	"pomu/fs"
)

type SegmentWriter struct {
	fs fs.FS
	id string
}

func NewSegmentWriter(fs fs.FS, id string) SegmentWriter {
	return SegmentWriter{fs, id}
}

func (writer *SegmentWriter) write(segmentID int, buffer []byte) error {
	segmentName := fmt.Sprintf("%03d.ts", segmentID)
	err := writer.fs.Write(path.Join(writer.id, segmentName), buffer)
	if err != nil {
		return err
	}
	log.Println("Wrote segment", segmentID)
	return nil
}
