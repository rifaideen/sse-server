[![Go Reference](https://pkg.go.dev/badge/github.com/rifaideen/sse-server.svg)](https://pkg.go.dev/github.com/rifaideen/sse-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/rifaideen/sse-server)](https://goreportcard.com/report/github.com/rifaideen/sse-server)

# SSE Server

A robust and efficient Server-Sent Events (SSE) server for Go applications.

The `sse-server` package simplifies the implementation of Server-Sent Events (SSE) in Go, allowing you to easily push real-time updates from your server to clients over HTTP. It provides a clean and straightforward API, handling the complexities of SSE for you.

## Features

- **Simple API:** Easy to use and integrate into your Go applications.
- **Concurrent Connections:** Handles multiple client connections efficiently.
- **Message Broadcasting:** Broadcasts messages to all connected clients.
- **Customizable Logging:** Utilizes`slog` for flexible logging.
- **Graceful Shutdown:** Supports graceful server shutdown.
- **Buffered Channels:** Uses buffered channels for efficient message handling.

## Concurrency Handling

The `sse-server` package is designed to handle multiple concurrent client connections efficiently and safely. To achieve this, the following concurrency mechanisms are employed:

* **Channels:** Channels are used for communication between goroutines, ensuring safe data transfer.
* **`sync.Map`:** `sync.Map` is used for concurrent access to the client connection map, minimizing contention.
* **Race Detection:** The package is regularly tested with the Go race detector (`go test -race`) to identify and prevent race conditions.

Extensive testing has been performed to ensure that the server handles high concurrency scenarios without data races or other concurrency-related issues.

## Installation

```sh
go get github.com/rifaideen/sse-server
```

## Usage

```go
package main

import (
    "log/slog"
    "os"

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

	fmt.Println("SSE server listening on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("HTTP server error", "error", err)
	}
}
```

## Examples

To explore practical examples of using the `sse-server` package, navigate to the `_examples` directory within the package.

## Feedback

- [Submit feedback](https://github.com/rifaideen/sse-server/issues/new)

## Disclaimer of Non-Liability

This project is provided **"as is"** and **without any express or implied warranties**, including, but not limited to, the implied warranties of merchantability and fitness for a particular purpose. In no event shall the authors or copyright holders be liable for any claim, damages or other liability, whether in an action of contract, tort or otherwise, arising from, out of or in connection with the software or the use or other dealings in the software.

## License

The `sse-server` package is licensed under the [Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0).


## Additional Resources

[SSE Documentation](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
