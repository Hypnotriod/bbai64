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

func BufferSizeFromTotal(total int) int {
	if total < 4 {
		return 0
	}
	return total/2 - 2
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

func (s *Streamer[T]) NewClient(buffSize int) *Client[T] {
	ch := make(chan *T, buffSize)
	c := &Client[T]{
		streamer: s,
		input:    ch,
		C:        ch,
	}
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		close(ch)
		return c
	}
	c.streamer.add <- c
	s.mu.Unlock()
	return c
}

func (s *Streamer[T]) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRunning
}

func (s *Streamer[T]) Broadcast(data *T) bool {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return false
	}
	s.broadcast <- data
	s.mu.Unlock()
	return true
}

func (s *Streamer[T]) Run() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = true
	s.mu.Unlock()
loop:
	for {
		select {
		case <-s.stop:
			for client := range s.clients {
				close(client.input)
			}
			clear(s.clients)
			break loop
		case client := <-s.add:
			s.clients[client] = true
		case client := <-s.remove:
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.input)
			}
		case chunk := <-s.broadcast:
			for client := range s.clients {
				client.input <- chunk
			}
		}
	}
}

func (s *Streamer[T]) Stop() bool {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return false
	}
	s.isRunning = false
	s.stop <- true
	s.mu.Unlock()
	return true
}
