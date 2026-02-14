package manager

import (
	"fmt"
	"sync"
)

type SSEBroker struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

var Broker = &SSEBroker{
	clients: make(map[chan string]struct{}),
}

func (b *SSEBroker) Subscribe() chan string {
	ch := make(chan string, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *SSEBroker) Unsubscribe(ch chan string) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

func (b *SSEBroker) Publish(requestID uint, ideName, appName, question string) {
	msg := fmt.Sprintf(`{"id":%d,"ide_name":%q,"app_name":%q,"question":%q}`, requestID, ideName, appName, question)
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}
