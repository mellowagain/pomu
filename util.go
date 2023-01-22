package main

import (
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"golang.org/x/exp/rand"
	"net/http"
)

// SerializeJson serializes `value` as Json into `writer`. If it fails, status code 500 Internal Server Error will be sent
func SerializeJson(writer http.ResponseWriter, value any) {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "    ")

	writer.Header().Set("Content-Type", "application/json")

	if err := encoder.Encode(value); err != nil {
		writer.Header().Set("Content-Type", "text/plain")

		sentry.CaptureException(err)
		http.Error(writer, "failed to serialize response to json", http.StatusInternalServerError)
	}
}

func RandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:length]
}
