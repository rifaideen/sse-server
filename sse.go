package sse

import (
	"errors"
	"log/slog"
	"sync"
)

// Quit message to close the client connection
const QUIT = "[[QUIT]]"

// Error returned when the logger is nil
var ErrNilLogger = errors.New("nil logger")

// SSE Server which contains the list of clients and channels to add / remove clients and a channel to receive notification.
//
// Upon receiving messages from Add / Remove clients, the list of clients will be updated accordingly.
//
// When any message is received on Notification channel, the message will be broadcasted to the list of clients available for notification.
//
// When the server is closed, the list of clients will be closed individually and make sure you are handling the client channel closure properly.
type Server struct {
	// List of clients connected to the server
	clients *sync.Map

	// Channel to close the server
	close chan bool

	// Flag to check if the server is closed
	closed bool

	// Close the server exactly once
	once sync.Once

	// Logger service
	logger *slog.Logger

	// Add new client to the server
	Add chan chan string

	// Remove client from the server
	Remove chan chan string

	// Send notification to all clients in the server
	Notification chan string
}

// Create new instance of SSE Server. Server is responsible for managing the list of clients and broadcasting messages to the clients.
//
// Example:
//
//	/**
//	 * Create new instance of SSE Server and start listening for new clients.
//	 *
//	 * It is recommended to create once in main() goroutine.
//	 */
//	server, _ := sse.New(logger)
//
//	// start listening for the client addition, removal and notification messages
//	go server.Listen()
//	defer server.Close()
//
//	/**
//	 * somewhere in your application, using the same sse server instance, you can
//	 * add, remove client and send notification
//	 */
//	// To add new client, send client channel to Add channel.
//	client := make(chan string, 100) // with buffer size 100
//	server.Add <- client
//
//	// To remove client, send client channel to Remove channel.
//	server.Remove <- client
//
//	// To receive notification, read from client channel. In your http handler, set text/event-stream and keep-alive headers to support http/2 SSE.
//	flusher := http.NewResponseController(w)
//
//	for message := range client {
//		fmt.FPrintf(w, "event: notification\ndata: %s\n\n", message)
//
//		// flush to immediately broadcast the message to the client (browser)
//		flusher.Flush()
//	}
//
//	// To send notification to all clients, send message to Notification channel.
//	server.Notification <- "Hello World!"
func New(logger *slog.Logger) (*Server, error) {
	if logger == nil {
		return nil, ErrNilLogger
	}

	return &Server{
		clients:      &sync.Map{},
		logger:       logger,
		close:        make(chan bool),
		Add:          make(chan chan string),
		Remove:       make(chan chan string),
		Notification: make(chan string, 1024),
	}, nil
}

// Add new client to the list of clients
func (server *Server) add(client chan string) {
	server.clients.Store(client, struct{}{})
}

// Remove client from the list of clients
func (server *Server) remove(client chan string) {
	server.clients.Delete(client)
}

// Broadcast message to all clients in the list
func (server *Server) broadcast(message string) {
	server.logger.Info("SSE Server: Broadcast", "message", message)

	server.clients.Range(func(client, value any) bool {
		select {
		case client.(chan string) <- message:
		default:
			server.logger.Debug("SSE Server: Client channel is full, dropping message", "message", message)
		}

		return true
	})
}

// Listen for incoming messages and perform the appropriate actions
//
// This method must be run as goroutine.
func (server *Server) Listen() {
	server.logger.Info("SSE Server: Listening")

	for {
		select {
		case <-server.close:
			server.cleanup()

			return
		case client := <-server.Add:
			server.add(client)
		case client := <-server.Remove:
			server.remove(client)
		case message := <-server.Notification:
			server.logger.Info("SSE Server: Broadcast", "message", message)
			server.broadcast(message)
		}
	}
}

// Close the sse server. Once closed, the server cannot be reopened.
//
// It is recommended to call this method in defer statement.
//
// You  can check if the server is closed by calling IsClosed() method.
//
// Upon closing the server, the server will broadcast QUIT message to all the clients and remove them from the server.
//
// The clients can handle the QUIT message and do the necessary cleanup and close the channel.
func (server *Server) Close() {
	server.once.Do(func() {
		close(server.close)
		server.closed = true
	})
}

// Check if the server is closed
func (server *Server) IsClosed() bool {
	return server.closed
}

// cleanup the server by broadcasting the clients about the closure then closing the channels and removing all the clients.
// This method is called when the server is closed.
func (server *Server) cleanup() {
	server.logger.Info("SSE Server: Cleaning up")

	// broadcast to all the connected client before disconnecting them
	server.broadcast(QUIT)
	// clear all the clients
	server.clients.Clear()

	// close all the channels
	close(server.Add)
	close(server.Remove)
	close(server.Notification)

	server.logger.Info("SSE Server: Cleaned up")
	server.logger.Info("SSE Server: Closed")
}
