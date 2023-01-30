package video

import (
	"context"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"

	"github.com/getsentry/sentry-go"
)

type Muxer struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	// Stderr is where to write stderr to
	Stderr io.Writer
}

func (w *Muxer) Start() error {
	span := sentry.StartSpan(context.Background(), "ffmpeg start muxer")
	defer span.Finish()

	cmd := exec.Command(
		os.Getenv("FFMPEG"),
		"-i", "pipe:0",
		"-c", "copy",
		"-movflags", "frag_keyframe+empty_moov",
		"-max_muxing_queue_size", "1024",
		"-bsf:a", "aac_adtstoasc",
		"-f", "mp4",
		"-hide_banner",
		"pipe:1")

	var err error
	w.stdin, err = cmd.StdinPipe()
	if err != nil {
		return err
	}
	w.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = w.Stderr
	w.cmd = cmd
	defer log.Println("Started ffmpeg")
	return w.cmd.Start()
}

func (w *Muxer) Write(p []byte) (int, error) {
	return w.stdin.Write(p)
}

func (w *Muxer) Read(p []byte) (int, error) {
	return w.stdout.Read(p)
}

// Close implements Closer
func (w *Muxer) Close() error {
	return w.stdin.Close()
}

func (w *Muxer) closeWrite() error {
	return w.stdin.Close()
}

func (w *Muxer) closeRead() error {
	return w.stdout.Close()
}

func (w *Muxer) Stop() {
	w.closeWrite()
}

var _ io.Writer = (*Muxer)(nil)
var _ io.Reader = (*Muxer)(nil)
