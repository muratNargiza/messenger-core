package messenger

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/gliedabrennung/messenger-core/internal/pkg/logger"
	"github.com/gliedabrennung/messenger-core/internal/domain"
)

const shardCount = 32

type Hub struct {
	shards  [shardCount]*shard
	done    chan struct{}
	MsgRepo domain.MessageRepository
}

type shard struct {
	clients    map[int64]*Client
	register   chan *Client
	unregister chan *Client
	direct     chan DirectMessage
	done       chan struct{}
}

type DirectMessage struct {
	From    int64  `json:"from"`
	To      int64  `json:"to"`
	Message string `json:"message"`
}

func NewHub(msgRepo domain.MessageRepository) *Hub {
	h := &Hub{
		done:    make(chan struct{}),
		MsgRepo: msgRepo,
	}
	for i := range h.shards {
		h.shards[i] = &shard{
			clients:    make(map[int64]*Client),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			direct:     make(chan DirectMessage, 256),
			done:       h.done,
		}
	}
	return h
}

func (h *Hub) getShard(userID int64) *shard {
	idx := userID % shardCount
	if idx < 0 {
		idx = -idx
	}
	return h.shards[idx]
}

func (h *Hub) Register(c *Client) {
	s := h.getShard(c.id)
	select {
	case s.register <- c:
	case <-h.done:
	}
}

func (h *Hub) Unregister(c *Client) {
	s := h.getShard(c.id)
	select {
	case s.unregister <- c:
	case <-h.done:
	}
}

func (h *Hub) Send(msg DirectMessage) bool {
	s := h.getShard(msg.To)
	select {
	case s.direct <- msg:
		return true
	case <-h.done:
		return false
	}
}

func (h *Hub) Done() <-chan struct{} {
	return h.done
}

func (h *Hub) Stop() {
	close(h.done)
}

func (h *Hub) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, s := range h.shards {
		wg.Add(1)
		go func(s *shard) {
			defer wg.Done()
			s.run(ctx)
		}(s)
	}
	wg.Wait()
	logger.Info("hub: shutdown complete")
}

func (s *shard) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.shutdown()
			return
		case <-s.done:
			s.shutdown()
			return
		case client := <-s.register:
			if old, ok := s.clients[client.id]; ok {
				close(old.send)
			}
			s.clients[client.id] = client
		case client := <-s.unregister:
			if c, ok := s.clients[client.id]; ok && c == client {
				delete(s.clients, client.id)
				close(client.send)
			}
		case msg := <-s.direct:
			if client, ok := s.clients[msg.To]; ok {
				msgBytes, err := json.Marshal(msg)
				if err != nil {
					logger.Errorf("hub: marshal direct message: %v", err)
					continue
				}
				select {
				case client.send <- msgBytes:
				default:
					close(client.send)
					delete(s.clients, client.id)
				}
			}
		}
	}
}

func (s *shard) shutdown() {
	for id, client := range s.clients {
		close(client.send)
		delete(s.clients, id)
	}
}
