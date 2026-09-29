package realtime

import "sync"

type Hub struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]struct{}
}

func New() *Hub {
	return &Hub{subscribers: make(map[chan []byte]struct{})}
}

func (h *Hub) Subscribe() (<-chan []byte, func()) {
	channel := make(chan []byte, 1)
	h.mu.Lock()
	h.subscribers[channel] = struct{}{}
	h.mu.Unlock()
	return channel, func() {
		h.mu.Lock()
		delete(h.subscribers, channel)
		h.mu.Unlock()
	}
}

func (h *Hub) Publish(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for subscriber := range h.subscribers {
		copyOfMessage := append([]byte(nil), message...)
		select {
		case subscriber <- copyOfMessage:
		default:
		}
	}
}
