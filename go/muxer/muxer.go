package muxer

import "sync"

type Client[T any] struct {
	muxer   *Muxer[T]
	Receive chan *T
}

func NewClient[T any](mux *Muxer[T]) *Client[T] {
	c := &Client[T]{
		muxer:   mux,
		Receive: make(chan *T),
	}
	c.muxer.add <- c
	return c
}

func (c *Client[T]) Close() {
	for {
		select {
		case <-c.Receive:
		case c.muxer.remove <- c:
			return
		}
	}
}

type Muxer[T any] struct {
	mu        sync.Mutex
	isRunning bool
	clients   map[*Client[T]]bool
	add       chan *Client[T]
	remove    chan *Client[T]
	broadcast chan *T
	stop      chan bool
}

func NewMuxer[T any](buffSize int) *Muxer[T] {
	mux := &Muxer[T]{
		clients:   make(map[*Client[T]]bool),
		add:       make(chan *Client[T]),
		remove:    make(chan *Client[T]),
		broadcast: make(chan *T, buffSize),
		stop:      make(chan bool),
	}
	go mux.run()
	return mux
}

func (m *Muxer[T]) Broadcast(data *T) bool {
	m.mu.Lock()
	if !m.isRunning {
		m.mu.Unlock()
		return false
	}
	m.broadcast <- data
	m.mu.Unlock()
	return true
}

func (m *Muxer[T]) run() {
	m.isRunning = true
	for {
		select {
		case <-m.stop:
			for client := range m.clients {
				close(client.Receive)
			}
			clear(m.clients)
			break
		case client := <-m.add:
			m.clients[client] = true
		case client := <-m.remove:
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				close(client.Receive)
			}
		case chunk := <-m.broadcast:
			for client := range m.clients {
				client.Receive <- chunk
			}
		}
	}
}

func (m *Muxer[T]) Stop() {
	m.mu.Lock()
	m.isRunning = false
	m.stop <- true
	m.mu.Unlock()
}
