package muxer

import "sync"

type Client[T any] struct {
	muxer *Muxer[T]
	C     chan *T
}

func (c *Client[T]) Close() {
	for {
		select {
		case <-c.C:
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
	return &Muxer[T]{
		clients:   make(map[*Client[T]]bool),
		add:       make(chan *Client[T]),
		remove:    make(chan *Client[T]),
		broadcast: make(chan *T, buffSize),
		stop:      make(chan bool),
	}
}

func (m *Muxer[T]) NewClient(buffSize int) *Client[T] {
	c := &Client[T]{
		muxer: m,
		C:     make(chan *T, buffSize),
	}
	c.muxer.add <- c
	return c
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

func (m *Muxer[T]) Run() {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return
	}
	m.isRunning = true
	m.mu.Unlock()
	for {
		select {
		case <-m.stop:
			for client := range m.clients {
				close(client.C)
			}
			clear(m.clients)
			break
		case client := <-m.add:
			m.clients[client] = true
		case client := <-m.remove:
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				close(client.C)
			}
		case chunk := <-m.broadcast:
			for client := range m.clients {
				client.C <- chunk
			}
		}
	}
}

func (m *Muxer[T]) Stop() bool {
	m.mu.Lock()
	if !m.isRunning {
		m.mu.Unlock()
		return false
	}
	m.isRunning = false
	m.stop <- true
	m.mu.Unlock()
	return true
}
