package webserver

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/tejzpr/rishvan-mcp/internal/config"
	"github.com/tejzpr/rishvan-mcp/internal/db"
	"github.com/tejzpr/rishvan-mcp/internal/manager"
)

var (
	startOnce sync.Once
	startErr  error
)

// EmbeddedFS will be set from main.go with the embedded frontend files
var EmbeddedFS fs.FS

func Start() error {
	startOnce.Do(func() {
		mux := http.NewServeMux()

		// API routes
		mux.HandleFunc("GET /api/requests", handleListRequests)
		mux.HandleFunc("GET /api/requests/{id}", handleGetRequest)
		mux.HandleFunc("POST /api/requests/{id}/respond", handleRespond)
		mux.HandleFunc("OPTIONS /api/", handleCORS)
		mux.HandleFunc("GET /api/events", handleSSE)
		mux.HandleFunc("GET /api/ide", handleIDE)

		// Serve embedded frontend
		if EmbeddedFS != nil {
			fileServer := http.FileServer(http.FS(EmbeddedFS))
			mux.Handle("/", fileServer)
		}

		ln, err := net.Listen("tcp", ":56234")
		if err != nil {
			startErr = err
			return
		}

		go func() {
			_ = http.Serve(ln, corsMiddleware(mux))
		}()
	})
	return startErr
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
	json.NewEncoder(w).Encode(map[string]string{"ide_name": config.IDEName})
}

func IsRunning() bool {
	conn, err := net.Dial("tcp", "localhost:56234")
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func handleListRequests(w http.ResponseWriter, r *http.Request) {
	database := db.Get()
	if database == nil {
		http.Error(w, "database not initialized", http.StatusInternalServerError)
		return
	}

	var requests []db.Request
	query := database.Where("ide_name = ?", config.IDEName).Order("created_at DESC")

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
	if err := database.Where("ide_name = ?", config.IDEName).First(&req, id).Error; err != nil {
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
