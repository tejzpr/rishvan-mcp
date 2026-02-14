package manager

import (
	"fmt"
	"sync"
	"time"

	"github.com/tejzpr/rishvan-mcp/internal/db"
)

type RequestManager struct {
	mu       sync.Mutex
	channels map[uint]chan string
}

var Instance = NewRequestManager()

// NewRequestManager creates a fresh RequestManager.
func NewRequestManager() *RequestManager {
	return &RequestManager{
		channels: make(map[uint]chan string),
	}
}

func (m *RequestManager) CreateRequest(ideName, appName, question string) (uint, <-chan string, error) {
	database := db.Get()
	if database == nil {
		return 0, nil, fmt.Errorf("database not initialized")
	}

	req := db.Request{
		IDEName:  ideName,
		AppName:  appName,
		Question: question,
		Status:   "pending",
	}
	if err := database.Create(&req).Error; err != nil {
		return 0, nil, fmt.Errorf("failed to create request: %w", err)
	}

	ch := make(chan string, 1)
	m.mu.Lock()
	m.channels[req.ID] = ch
	m.mu.Unlock()

	return req.ID, ch, nil
}

func (m *RequestManager) RespondToRequest(id uint, response string) error {
	database := db.Get()
	if database == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now()
	result := database.Model(&db.Request{}).Where("id = ? AND status = ?", id, "pending").Updates(map[string]interface{}{
		"response":     response,
		"status":       "responded",
		"responded_at": &now,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to update request: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("request %d not found or already responded", id)
	}

	m.mu.Lock()
	ch, ok := m.channels[id]
	if ok {
		delete(m.channels, id)
	}
	m.mu.Unlock()

	if ok {
		ch <- response
		close(ch)
	}

	return nil
}
