package main

import (
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func StartRecording(request VideoRequest, try int32) {
	span := sentry.StartSpan(context.Background(), "youtube-dl get playlist", sentry.TransactionName(fmt.Sprintf("youtube-dl get playlist %s", request.VideoUrl)))

	output := new(strings.Builder)

	cmd := exec.Command(os.Getenv("YOUTUBE_DL"), "-f", string(request.Quality), "-g", request.VideoUrl)
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Run(); err != nil {
		if strings.Contains(output.String(), "ERROR: This live event will begin in") {
			if try <= 120 {
				// Event has not yet started; try again in a minute
				time.Sleep(1 * time.Minute)
				StartRecording(request, try+1)
				return
			} else {
				sentry.CaptureMessage(fmt.Sprintf("Tried to record %s for two hours but livestream didn't start. Giving up", request.VideoUrl))
				log.Printf("Tried to record %s for two hours but livestream didn't start. Giving up\n", request.VideoUrl)
				return
			}
		} else {
			sentry.CaptureException(err)

			log.Printf("cannot run youtube-dl: %s\n", err)
			return
		}
	}

	span.Finish()

	stringOutput := output.String()

	if !strings.HasSuffix(stringOutput, ".m3u8") {
		log.Printf("Expected m3u8 output, received %s\n", stringOutput)
		return
	}

	// We have the playlist url, pass it to the recorder
}
