package muxer

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
	clients   map[*Client[T]]bool
	add       chan *Client[T]
	remove    chan *Client[T]
	Broadcast chan *T
}

func NewMuxer[T any](buffSize int) *Muxer[T] {
	return &Muxer[T]{
		clients:   make(map[*Client[T]]bool),
		add:       make(chan *Client[T]),
		remove:    make(chan *Client[T]),
		Broadcast: make(chan *T, buffSize),
	}
}

func (m *Muxer[T]) Run() {
	for {
		select {
		case client := <-m.add:
			m.clients[client] = true
		case client := <-m.remove:
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				close(client.Receive)
			}
		case chunk := <-m.Broadcast:
			for client := range m.clients {
				client.Receive <- chunk
			}
		}
	}
}
