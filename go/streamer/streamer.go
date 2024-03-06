package streamer

import "sync"

type Client[T any] struct {
	streamer *Streamer[T]
	input    chan<- *T
	C        <-chan *T
}

func (c *Client[T]) Close() {
	for {
		select {
		case _, ok := <-c.C:
			if !ok {
				return
			}
		case c.streamer.remove <- c:
			return
		}
	}
}

type Streamer[T any] struct {
	mu        sync.Mutex
	isRunning bool
	clients   map[*Client[T]]bool
	add       chan *Client[T]
	remove    chan *Client[T]
	broadcast chan *T
	stop      chan bool
}

func NewStreamer[T any](buffSize int) *Streamer[T] {
	return &Streamer[T]{
		clients:   make(map[*Client[T]]bool),
		add:       make(chan *Client[T]),
		remove:    make(chan *Client[T]),
		broadcast: make(chan *T, buffSize),
		stop:      make(chan bool),
	}
}

func (m *Streamer[T]) NewClient(buffSize int) *Client[T] {
	ch := make(chan *T, buffSize)
	c := &Client[T]{
		streamer: m,
		input:    ch,
		C:        ch,
	}
	c.streamer.add <- c
	return c
}

func (m *Streamer[T]) Broadcast(data *T) bool {
	m.mu.Lock()
	if !m.isRunning {
		m.mu.Unlock()
		return false
	}
	m.broadcast <- data
	m.mu.Unlock()
	return true
}

func (m *Streamer[T]) Run() {
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
				close(client.input)
			}
			clear(m.clients)
			break
		case client := <-m.add:
			m.clients[client] = true
		case client := <-m.remove:
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				close(client.input)
			}
		case chunk := <-m.broadcast:
			for client := range m.clients {
				client.input <- chunk
			}
		}
	}
}

func (m *Streamer[T]) Stop() bool {
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
