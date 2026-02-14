package manager

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/tejzpr/rishvan-mcp/internal/db"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	d, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := d.AutoMigrate(&db.Request{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	db.InitWithDB(d)
}

func newTestManager() *RequestManager {
	return &RequestManager{
		channels: make(map[uint]chan string),
	}
}

func TestCreateRequest(t *testing.T) {
	setupTestDB(t)
	m := newTestManager()

	id, ch, err := m.CreateRequest("test-ide", "my-app", "What should I do?")
	if err != nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero request ID")
	}
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	// Verify DB record
	var req db.Request
	if err := db.Get().First(&req, id).Error; err != nil {
		t.Fatalf("failed to fetch from DB: %v", err)
	}
	if req.AppName != "my-app" {
		t.Errorf("expected app_name 'my-app', got %q", req.AppName)
	}
	if req.Question != "What should I do?" {
		t.Errorf("expected question 'What should I do?', got %q", req.Question)
	}
	if req.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", req.Status)
	}
}

func TestRespondToRequest(t *testing.T) {
	setupTestDB(t)
	m := newTestManager()

	id, ch, err := m.CreateRequest("test-ide", "app", "Pick a color")
	if err != nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}

	// Respond in a goroutine
	go func() {
		time.Sleep(50 * time.Millisecond)
		if err := m.RespondToRequest(id, "blue"); err != nil {
			t.Errorf("RespondToRequest failed: %v", err)
		}
	}()

	// Block on channel
	select {
	case resp := <-ch:
		if resp != "blue" {
			t.Errorf("expected 'blue', got %q", resp)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for response")
	}

	// Verify DB updated
	var req db.Request
	db.Get().First(&req, id)
	if req.Status != "responded" {
		t.Errorf("expected status 'responded', got %q", req.Status)
	}
	if req.Response != "blue" {
		t.Errorf("expected response 'blue', got %q", req.Response)
	}
	if req.RespondedAt == nil {
		t.Error("expected responded_at to be set")
	}
}

func TestRespondToAlreadyRespondedRequest(t *testing.T) {
	setupTestDB(t)
	m := newTestManager()

	id, _, err := m.CreateRequest("test-ide", "app", "question")
	if err != nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}
	if err := m.RespondToRequest(id, "first"); err != nil {
		t.Fatalf("first respond failed: %v", err)
	}

	// Second respond should fail
	err = m.RespondToRequest(id, "second")
	if err == nil {
		t.Fatal("expected error when responding to already-responded request")
	}
}

func TestRespondToNonExistentRequest(t *testing.T) {
	setupTestDB(t)
	m := newTestManager()

	err := m.RespondToRequest(99999, "hello")
	if err == nil {
		t.Fatal("expected error for non-existent request")
	}
}

func TestConcurrentRespond(t *testing.T) {
	setupTestDB(t)
	m := newTestManager()

	const n = 5

	// Create all requests sequentially
	type entry struct {
		id uint
		ch <-chan string
	}
	entries := make([]entry, n)
	for i := 0; i < n; i++ {
		id, ch, err := m.CreateRequest("test-ide", "concurrent-app", "question")
		if err != nil {
			t.Fatalf("create %d failed: %v", i, err)
		}
		entries[i] = entry{id: id, ch: ch}
	}

	// Respond sequentially in a goroutine (SQLite serializes writes),
	// but verify all channels unblock correctly.
	go func() {
		for i, e := range entries {
			if err := m.RespondToRequest(e.id, fmt.Sprintf("answer-%d", i)); err != nil {
				t.Errorf("respond %d failed: %v", i, err)
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			select {
			case resp := <-entries[i].ch:
				expected := fmt.Sprintf("answer-%d", i)
				if resp != expected {
					t.Errorf("expected %q, got %q", expected, resp)
				}
			case <-time.After(5 * time.Second):
				t.Errorf("timed out on request %d", i)
			}
		}(i)
	}

	wg.Wait()
}
