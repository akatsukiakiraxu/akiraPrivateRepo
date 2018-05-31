package core

import (
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"golang.org/x/net/websocket"
)

type WSServer struct {
	uriPattern       string
	isListening      int32
	connectedHandler func(c *WSClient)
	clients          map[*WSClient]*WSClient
	clientsLock      sync.RWMutex
}

func NewWSServer(uriPattern string, connectedHandler func(c *WSClient)) *WSServer {
	return &WSServer{
		uriPattern:       uriPattern,
		connectedHandler: connectedHandler,
		clients:          make(map[*WSClient]*WSClient, 2),
		clientsLock:      sync.RWMutex{},
	}
}
func (s *WSServer) AddClient(c *WSClient) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()
	s.clients[c] = c
}
func (s *WSServer) RemoveClient(c *WSClient) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()
	delete(s.clients, c)
}
func (s *WSServer) IsListening() bool {
	return s.isListening != 0
}
func (s *WSServer) Listen() {
	if !atomic.CompareAndSwapInt32(&s.isListening, 0, 1) {
		return
	}

	onConnected := func(ws *websocket.Conn) {
		defer ws.Close()
		client := NewWSClient(ws, s)
		s.AddClient(client)
		if s.connectedHandler != nil {
			s.connectedHandler(client)
		}
		client.Listen()
	}
	http.HandleFunc(s.uriPattern, func(w http.ResponseWriter, req *http.Request) {
		hs := websocket.Server{Handler: websocket.Handler(onConnected)}
		hs.ServeHTTP(w, req)
	})
}
func (s *WSServer) Stop() {
	s.clientsLock.RLock()
	defer s.clientsLock.RUnlock()
	for _, c := range s.clients {
		c.Stop()
	}
}
func (s *WSServer) Write(data interface{}) {
	s.clientsLock.RLock()
	defer s.clientsLock.RUnlock()
	for _, c := range s.clients {
		c.Write(data)
	}
}

type WSClient struct {
	conn    *websocket.Conn
	writeCh chan interface{}
	stopCh  chan struct{}
	doneCh  chan struct{}
}

func NewWSClient(conn *websocket.Conn, s *WSServer) *WSClient {
	return &WSClient{
		conn:    conn,
		writeCh: make(chan interface{}, 10),
		stopCh:  make(chan struct{}),
	}
}
func (c *WSClient) Write(data interface{}) {
	c.writeCh <- data

}
func (c *WSClient) Listen() {
	stopSendCh := make(chan struct{})
	go func() {
		defer func() {
			c.doneCh <- struct{}{}
		}()
		for {
			select {
			case data := <-c.writeCh:
				websocket.Message.Send(c.conn, data)
			case <-c.stopCh:
				return
			}
		}
	}()
	func() {
		for {
			select {
			case <-c.stopCh:
				return
			default:
				var s string
				switch err := websocket.Message.Receive(c.conn, &s); err {
				case io.EOF:
					log.Printf("Closed")
					stopSendCh <- struct{}{}
					return
				case nil:
					break
				default:
					log.Printf("Error: %s", err.Error())
				}
			}
		}
	}()
	<-c.doneCh

	// Read remaining data from write channel and discard them.
	for {
		select {
		case _, ok := <-c.writeCh:
			if !ok {
				return
			}
		}
	}
}
func (c *WSClient) Stop() {
	c.stopCh <- struct{}{}
}
