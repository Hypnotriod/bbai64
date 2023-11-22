package muxer

const CHUNK_SIZE = 4096

type Chunk struct {
	Data [CHUNK_SIZE]byte
	Size int
}

type Client struct {
	muxer *Muxer
	Send  chan *Chunk
}

func NewClient(mux *Muxer) *Client {
	c := &Client{
		muxer: mux,
		Send:  make(chan *Chunk),
	}
	c.muxer.add <- c
	return c
}

func (c *Client) Close() {
	for {
		select {
		case <-c.Send:
		case c.muxer.remove <- c:
			return
		}
	}
}

type Muxer struct {
	clients   map[*Client]bool
	add       chan *Client
	remove    chan *Client
	Broadcast chan *Chunk
}

func NewMuxer(buffSize int) *Muxer {
	return &Muxer{
		clients:   make(map[*Client]bool),
		add:       make(chan *Client),
		remove:    make(chan *Client),
		Broadcast: make(chan *Chunk, buffSize),
	}
}

func (m *Muxer) Run() {
	for {
		select {
		case client := <-m.add:
			m.clients[client] = true
		case client := <-m.remove:
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				close(client.Send)
			}
		case chunk := <-m.Broadcast:
			for client := range m.clients {
				client.Send <- chunk
			}
		}
	}
}
