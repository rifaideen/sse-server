package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	sse "github.com/rifaideen/sse-server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server, err := sse.New(logger)

	if err != nil {
		logger.Error("failed to create sse server", "error", err)
		return
	}

	go server.Listen()
	defer server.Close()

	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)

		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		// create buffered channel to receive notifications and add it to the server
		chNotification := make(chan string, 10)
		server.Add <- chNotification

		defer func() {
			// remove the channel from the server and close the channel
			server.Remove <- chNotification
			close(chNotification)
		}()

		for {
			select {
			case message := <-chNotification:
				// Quit signal received, exit the loop
				if message == sse.QUIT {
					return
				}

				// send message to client
				fmt.Fprintf(w, "data: %s\n\n", message)
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

	// Send server time updates every 2 seconds
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			server.Notification <- fmt.Sprintf("Server time: %s", time.Now().Format(time.RFC3339))
		}
	}()

	logger.Info("SSE server listening on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("HTTP server error", "error", err)
	}
}
