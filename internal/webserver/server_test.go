package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tejzpr/rishvan-mcp/internal/config"
	"github.com/tejzpr/rishvan-mcp/internal/db"
	"github.com/tejzpr/rishvan-mcp/internal/manager"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	config.IDEName = "test-ide"
	d, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
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

func seedRequests(t *testing.T) {
	t.Helper()
	d := db.Get()
	d.Create(&db.Request{IDEName: "test-ide", AppName: "app-a", Question: "q1", Status: "pending"})
	d.Create(&db.Request{IDEName: "test-ide", AppName: "app-b", Question: "q2", Status: "pending"})
	d.Create(&db.Request{IDEName: "test-ide", AppName: "app-a", Question: "q3", Status: "responded", Response: "done"})
	d.Create(&db.Request{IDEName: "other-ide", AppName: "app-a", Question: "q4", Status: "pending"})
}

func TestHandleListRequests(t *testing.T) {
	setupTestDB(t)
	seedRequests(t)

	req := httptest.NewRequest("GET", "/api/requests", nil)
	w := httptest.NewRecorder()
	handleListRequests(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var requests []db.Request
	if err := json.NewDecoder(w.Body).Decode(&requests); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(requests) != 3 {
		t.Errorf("expected 3 requests (filtered by IDE), got %d", len(requests))
	}
	// Should be ordered newest first
	if requests[0].Question != "q3" {
		t.Errorf("expected newest first (q3), got %q", requests[0].Question)
	}
}

func TestHandleListRequestsFilterByApp(t *testing.T) {
	setupTestDB(t)
	seedRequests(t)

	req := httptest.NewRequest("GET", "/api/requests?app_name=app-a", nil)
	w := httptest.NewRecorder()
	handleListRequests(w, req)

	var requests []db.Request
	json.NewDecoder(w.Body).Decode(&requests)
	if len(requests) != 2 {
		t.Errorf("expected 2 app-a requests, got %d", len(requests))
	}
	for _, r := range requests {
		if r.AppName != "app-a" {
			t.Errorf("expected app_name 'app-a', got %q", r.AppName)
		}
	}
}

func TestHandleGetRequest(t *testing.T) {
	setupTestDB(t)
	db.Get().Create(&db.Request{IDEName: "test-ide", AppName: "app", Question: "hello", Status: "pending"})

	req := httptest.NewRequest("GET", "/api/requests/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	handleGetRequest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result db.Request
	json.NewDecoder(w.Body).Decode(&result)
	if result.Question != "hello" {
		t.Errorf("expected question 'hello', got %q", result.Question)
	}
}

func TestHandleGetRequestNotFound(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest("GET", "/api/requests/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()
	handleGetRequest(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetRequestInvalidID(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest("GET", "/api/requests/abc", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()
	handleGetRequest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleRespondSuccess(t *testing.T) {
	setupTestDB(t)
	origInstance := manager.Instance
	manager.Instance = manager.NewRequestManager()
	defer func() { manager.Instance = origInstance }()

	id, ch, err := manager.Instance.CreateRequest("test-ide", "app", "question?")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	body := strings.NewReader(`{"response":"my answer"}`)
	req := httptest.NewRequest("POST", "/api/requests/1/respond", body)
	req.SetPathValue("id", fmt.Sprintf("%d", id))
	w := httptest.NewRecorder()
	handleRespond(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Channel should have received the response
	select {
	case resp := <-ch:
		if resp != "my answer" {
			t.Errorf("expected 'my answer', got %q", resp)
		}
	default:
		t.Error("expected response on channel")
	}
}

func TestHandleRespondEmptyBody(t *testing.T) {
	setupTestDB(t)

	body := strings.NewReader(`{"response":""}`)
	req := httptest.NewRequest("POST", "/api/requests/1/respond", body)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	handleRespond(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty response, got %d", w.Code)
	}
}

func TestHandleRespondInvalidJSON(t *testing.T) {
	setupTestDB(t)

	body := strings.NewReader(`not json`)
	req := httptest.NewRequest("POST", "/api/requests/1/respond", body)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	handleRespond(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestHandleRespondInvalidID(t *testing.T) {
	setupTestDB(t)

	body := strings.NewReader(`{"response":"ok"}`)
	req := httptest.NewRequest("POST", "/api/requests/xyz/respond", body)
	req.SetPathValue("id", "xyz")
	w := httptest.NewRecorder()
	handleRespond(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid ID, got %d", w.Code)
	}
}

func TestHandleIDE(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest("GET", "/api/ide", nil)
	w := httptest.NewRecorder()
	handleIDE(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)
	if result["ide_name"] != "test-ide" {
		t.Errorf("expected ide_name 'test-ide', got %q", result["ide_name"])
	}
}

func TestIDEIsolation(t *testing.T) {
	setupTestDB(t)
	seedRequests(t)

	// Default config is "test-ide", should see 3 requests
	req := httptest.NewRequest("GET", "/api/requests", nil)
	w := httptest.NewRecorder()
	handleListRequests(w, req)

	var requests []db.Request
	json.NewDecoder(w.Body).Decode(&requests)
	if len(requests) != 3 {
		t.Errorf("expected 3 test-ide requests, got %d", len(requests))
	}

	// Switch to other-ide, should see 1 request
	config.IDEName = "other-ide"
	defer func() { config.IDEName = "test-ide" }()

	req2 := httptest.NewRequest("GET", "/api/requests", nil)
	w2 := httptest.NewRecorder()
	handleListRequests(w2, req2)

	var requests2 []db.Request
	json.NewDecoder(w2.Body).Decode(&requests2)
	if len(requests2) != 1 {
		t.Errorf("expected 1 other-ide request, got %d", len(requests2))
	}
	if requests2[0].Question != "q4" {
		t.Errorf("expected question 'q4', got %q", requests2[0].Question)
	}
}

func TestCORSMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware(inner)

	req := httptest.NewRequest("GET", "/api/requests", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS origin header")
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected CORS methods header")
	}
}
