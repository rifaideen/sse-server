package sse_test

import (
	"log/slog"
	"os"

	"sync"
	"testing"

	sse "github.com/rifaideen/sse-server"
	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	server, err := sse.New(nil)
	assert.Nil(t, server)
	assert.EqualError(t, err, sse.ErrNilLogger.Error())

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	server, err = sse.New(logger)
	assert.NoError(t, err)

	go server.Listen()
	defer server.Close()

	wg := sync.WaitGroup{}
	wg.Add(1000 + 1) // 1000 clients + 1 notification goroutine

	for range 1000 {
		notification := make(chan string, 4)
		server.Add <- notification

		go func() {
			defer wg.Done()

			for message := range notification {
				if message == sse.QUIT {
					close(notification)
					return
				}

				logger.Info(message)
			}
		}()
	}

	go func() {
		defer wg.Done()
		defer server.Close()

		for range 5 {
			server.Notification <- "message"
		}
	}()

	wg.Wait()
}

func TestOperations(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	server, err := sse.New(logger)
	assert.NoError(t, err)

	go server.Listen()

	wg := sync.WaitGroup{}
	wg.Add(1)

	notification := make(chan string, 100)
	server.Add <- notification

	go func() {
		defer wg.Done()
		message := <-notification
		assert.Equal(t, message, "message")

		server.Remove <- notification
	}()

	server.Notification <- "message"

	assert.False(t, server.IsClosed())

	wg.Wait()
	server.Close()

	assert.True(t, server.IsClosed())
}

func BenchmarkServer(b *testing.B) {
	// stop the timer before the server setup is completed
	b.StopTimer()

	logger := slog.New(slog.DiscardHandler)

	server, _ := sse.New(logger)
	go server.Listen()
	defer server.Close()

	// start the timer after the server setup is completed
	b.StartTimer()

	wg := sync.WaitGroup{}
	wg.Add(b.N + 1) // b.N clients + 1 notification goroutine

	// do not use b.Loop it will cause negative waitgroup error
	for range b.N {
		notification := make(chan string, 100)
		server.Add <- notification

		go func() {
			defer wg.Done()

			for message := range notification {
				if message == sse.QUIT {
					close(notification)
					return
				}

				logger.Info(message)
			}
		}()
	}

	go func() {
		defer wg.Done()
		defer server.Close()

		for range 5 {
			server.Notification <- "message"
		}
	}()

	wg.Wait()
}
