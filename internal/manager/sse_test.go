package manager

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSSEBrokerSubscribeUnsubscribe(t *testing.T) {
	b := &SSEBroker{
		clients: make(map[chan string]struct{}),
	}

	ch := b.Subscribe()
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	b.mu.RLock()
	count := len(b.clients)
	b.mu.RUnlock()
	if count != 1 {
		t.Errorf("expected 1 client, got %d", count)
	}

	b.Unsubscribe(ch)

	b.mu.RLock()
	count = len(b.clients)
	b.mu.RUnlock()
	if count != 0 {
		t.Errorf("expected 0 clients after unsubscribe, got %d", count)
	}
}

func TestSSEBrokerPublish(t *testing.T) {
	b := &SSEBroker{
		clients: make(map[chan string]struct{}),
	}

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Publish(42, "test-ide", "test-app", "What color?")

	select {
	case msg := <-ch:
		var data struct {
			ID       int    `json:"id"`
			IDEName  string `json:"ide_name"`
			AppName  string `json:"app_name"`
			Question string `json:"question"`
		}
		if err := json.Unmarshal([]byte(msg), &data); err != nil {
			t.Fatalf("failed to parse message: %v", err)
		}
		if data.ID != 42 {
			t.Errorf("expected id 42, got %d", data.ID)
		}
		if data.IDEName != "test-ide" {
			t.Errorf("expected ide_name 'test-ide', got %q", data.IDEName)
		}
		if data.AppName != "test-app" {
			t.Errorf("expected app_name 'test-app', got %q", data.AppName)
		}
		if data.Question != "What color?" {
			t.Errorf("expected question 'What color?', got %q", data.Question)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestSSEBrokerPublishMultipleClients(t *testing.T) {
	b := &SSEBroker{
		clients: make(map[chan string]struct{}),
	}

	ch1 := b.Subscribe()
	ch2 := b.Subscribe()
	ch3 := b.Subscribe()
	defer b.Unsubscribe(ch1)
	defer b.Unsubscribe(ch2)
	defer b.Unsubscribe(ch3)

	b.Publish(1, "test-ide", "app", "hello")

	for i, ch := range []chan string{ch1, ch2, ch3} {
		select {
		case msg := <-ch:
			if msg == "" {
				t.Errorf("client %d received empty message", i)
			}
		case <-time.After(time.Second):
			t.Errorf("client %d timed out", i)
		}
	}
}

func TestSSEBrokerPublishNoClients(t *testing.T) {
	b := &SSEBroker{
		clients: make(map[chan string]struct{}),
	}
	// Should not panic
	b.Publish(1, "test-ide", "app", "hello")
}

func TestSSEBrokerPublishDropsWhenFull(t *testing.T) {
	b := &SSEBroker{
		clients: make(map[chan string]struct{}),
	}

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// Fill the channel buffer (capacity 16)
	for i := 0; i < 20; i++ {
		b.Publish(uint(i), "test-ide", "app", "msg")
	}

	// Should have 16 messages (buffer size), rest dropped
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 16 {
		t.Errorf("expected 16 buffered messages, got %d", count)
	}
}
