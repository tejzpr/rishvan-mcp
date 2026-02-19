package webserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// RemoteCreateRequest sends a question to the primary rishvan-mcp server
// via HTTP and returns the created request ID.
func RemoteCreateRequest(sourceName, appName, question string) (uint, error) {
	payload, _ := json.Marshal(map[string]string{
		"source_name": sourceName,
		"app_name":    appName,
		"question":    question,
	})

	resp, err := http.Post(
		fmt.Sprintf("%s/api/requests", BaseURL),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to reach primary server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("primary server returned status %d", resp.StatusCode)
	}

	var result struct {
		ID uint `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}
	return result.ID, nil
}

// RemotePollResponse polls the primary server until the request is responded
// to or the context is cancelled. Returns the human's response text.
func RemotePollResponse(ctx context.Context, reqID uint) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			resp, err := client.Get(fmt.Sprintf("%s/api/requests/%d/poll", BaseURL, reqID))
			if err != nil {
				continue // transient error, retry
			}

			var result struct {
				ID       uint   `json:"id"`
				Status   string `json:"status"`
				Response string `json:"response"`
			}
			decErr := json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()
			if decErr != nil {
				continue
			}

			if result.Status == "responded" && result.Response != "" {
				return result.Response, nil
			}
		}
	}
}
