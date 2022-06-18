package main

import (
	"log"
	"os/exec"
	"strings"
	"time"
)

func StartRecording(request VideoRequest, try int32) {
	output := new(strings.Builder)

	cmd := exec.Command("youtube-dl", "-f", string(request.Quality), "-g", request.VideoUrl)
	cmd.Stdout = output

	if err := cmd.Run(); err != nil {
		log.Println("cannot run")
		return
	}

	stringOutput := output.String()

	if strings.HasPrefix(stringOutput, "ERROR: This live event") {
		if try <= 120 {
			// Event has not yet started; try again in a minute
			time.Sleep(1 * time.Minute)
			StartRecording(request, try+1)
			return
		} else {
			log.Printf("Tried to record %s for two hours but livestream didn't start. Giving up\n", request.VideoUrl)
			return
		}
	}

	if !strings.HasSuffix(stringOutput, ".m3u8") {
		log.Printf("Expected m3u8 output, received %s\n", stringOutput)
		return
	}

	// We have the playlist url, pass it to the recorder
}
