package services

import "sync"

type NotificationEvent struct {
	Role    string `json:"role"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type NotificationHub struct {
	mu      sync.RWMutex
	clients map[string]map[chan NotificationEvent]struct{}
}

func NewNotificationHub() *NotificationHub {
	return &NotificationHub{
		clients: make(map[string]map[chan NotificationEvent]struct{}),
	}
}

func (h *NotificationHub) Subscribe(role string) chan NotificationEvent {
	ch := make(chan NotificationEvent, 8)

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[role] == nil {
		h.clients[role] = make(map[chan NotificationEvent]struct{})
	}
	h.clients[role][ch] = struct{}{}

	return ch
}

func (h *NotificationHub) Unsubscribe(role string, ch chan NotificationEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[role] != nil {
		delete(h.clients[role], ch)
	}
	close(ch)
}

func (h *NotificationHub) Publish(event NotificationEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.clients[event.Role] {
		select {
		case ch <- event:
		default:
		}
	}
}
