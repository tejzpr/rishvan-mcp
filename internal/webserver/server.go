package webserver

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/tejzpr/rishvan-mcp/internal/config"
	"github.com/tejzpr/rishvan-mcp/internal/db"
	"github.com/tejzpr/rishvan-mcp/internal/manager"
)

const (
	Port        = 56234
	BaseURL     = "http://localhost:56234"
	healthMagic = "rishvan-mcp-ok"
)

var (
	startOnce sync.Once
	startErr  error
	// IsPrimary is true when this process owns the web server.
	IsPrimary bool
)

// EmbeddedFS will be set from main.go with the embedded frontend files
var EmbeddedFS fs.FS

// Start tries to bind the port. If the port is already held by another
// rishvan-mcp process it sets IsPrimary=false and returns nil so the
// caller can fall back to the HTTP-based remote client.
func Start() error {
	startOnce.Do(func() {
		// Try to bind the port first
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", Port))
		if err != nil {
			// Port taken – check if it is another rishvan-mcp instance
			if isRishvanServer() {
				IsPrimary = false
				return
			}
			startErr = fmt.Errorf("port %d in use by unknown process: %w", Port, err)
			return
		}

		IsPrimary = true

		mux := http.NewServeMux()

		// API routes
		mux.HandleFunc("GET /api/health", handleHealth)
		mux.HandleFunc("GET /api/requests", handleListRequests)
		mux.HandleFunc("GET /api/requests/{id}", handleGetRequest)
		mux.HandleFunc("POST /api/requests", handleCreateRequest)
		mux.HandleFunc("POST /api/requests/{id}/respond", handleRespond)
		mux.HandleFunc("GET /api/requests/{id}/poll", handlePollRequest)
		mux.HandleFunc("OPTIONS /api/", handleCORS)
		mux.HandleFunc("GET /api/events", handleSSE)
		mux.HandleFunc("GET /api/ide", handleIDE)

		// Serve embedded frontend
		if EmbeddedFS != nil {
			fileServer := http.FileServer(http.FS(EmbeddedFS))
			mux.Handle("/", fileServer)
		}

		go func() {
			_ = http.Serve(ln, corsMiddleware(mux))
		}()
	})
	return startErr
}

// isRishvanServer checks whether the process on Port is a rishvan-mcp server.
func isRishvanServer() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/api/health", BaseURL))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false
	}
	return body.Status == healthMagic
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": healthMagic})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		next.ServeHTTP(w, r)
	})
}

func handleCORS(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func handleIDE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"source_name": config.SourceName})
}

func IsRunning() bool {
	return isRishvanServer()
}

// handleCreateRequest allows a secondary (remote) IDE instance to create a
// request via HTTP. The primary instance stores it in the DB and manager.
func handleCreateRequest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SourceName string `json:"source_name"`
		AppName    string `json:"app_name"`
		Question   string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Question == "" || body.AppName == "" || body.SourceName == "" {
		http.Error(w, "source_name, app_name and question are required", http.StatusBadRequest)
		return
	}

	reqID, _, err := manager.Instance.CreateRequest(body.SourceName, body.AppName, body.Question)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Notify frontend via SSE
	manager.Broker.Publish(reqID, body.SourceName, body.AppName, body.Question)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": reqID})
}

// handlePollRequest lets a secondary instance poll until a request is responded to.
func handlePollRequest(w http.ResponseWriter, r *http.Request) {
	database := db.Get()
	if database == nil {
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req db.Request
	if err := database.First(&req, id).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       req.ID,
		"status":   req.Status,
		"response": req.Response,
	})
}

func handleListRequests(w http.ResponseWriter, r *http.Request) {
	database := db.Get()
	if database == nil {
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	var requests []db.Request
	query := database.Order("created_at DESC")

	// Filter by source_name if provided, otherwise show all
	if sourceName := r.URL.Query().Get("source_name"); sourceName != "" {
		query = query.Where("source_name = ?", sourceName)
	}

	if appName := r.URL.Query().Get("app_name"); appName != "" {
		query = query.Where("app_name = ?", appName)
	}

	if err := query.Find(&requests).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requests)
}

func handleGetRequest(w http.ResponseWriter, r *http.Request) {
	database := db.Get()
	if database == nil {
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req db.Request
	if err := database.First(&req, id).Error; err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req)
}

func handleRespond(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var body struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Response == "" {
		http.Error(w, "response cannot be empty", http.StatusBadRequest)
		return
	}

	if err := manager.Instance.RespondToRequest(uint(id), body.Response); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := manager.Broker.Subscribe()
	defer manager.Broker.Unsubscribe(ch)

	// Send initial keepalive
	fmt.Fprintf(w, ": keepalive\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: new-request\ndata: %s\n\n", msg)
			flusher.Flush()
		}
	}
}
