package db

import (
	"testing"
)

func TestCreateAndFetchRequest(t *testing.T) {
	setupTestDB(t)

	req := Request{
		SourceName: "test-ide",
		AppName:    "test-app",
		Question:   "What color?",
		Status:     "pending",
	}
	if err := instance.Create(&req).Error; err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if req.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}

	var fetched Request
	if err := instance.First(&fetched, req.ID).Error; err != nil {
		t.Fatalf("failed to fetch request: %v", err)
	}
	if fetched.AppName != "test-app" {
		t.Errorf("expected app_name 'test-app', got %q", fetched.AppName)
	}
	if fetched.Question != "What color?" {
		t.Errorf("expected question 'What color?', got %q", fetched.Question)
	}
	if fetched.Status != "pending" {
		t.Errorf("expected status 'pending', got %q", fetched.Status)
	}
}

func TestUpdateRequestResponse(t *testing.T) {
	setupTestDB(t)

	req := Request{
		SourceName: "test-ide",
		AppName:    "app-2",
		Question:   "Pick a number",
		Status:     "pending",
	}
	instance.Create(&req)

	result := instance.Model(&Request{}).Where("id = ? AND status = ?", req.ID, "pending").Updates(map[string]interface{}{
		"response": "42",
		"status":   "responded",
	})
	if result.Error != nil {
		t.Fatalf("update failed: %v", result.Error)
	}
	if result.RowsAffected != 1 {
		t.Fatalf("expected 1 row affected, got %d", result.RowsAffected)
	}

	var fetched Request
	instance.First(&fetched, req.ID)
	if fetched.Status != "responded" {
		t.Errorf("expected status 'responded', got %q", fetched.Status)
	}
	if fetched.Response != "42" {
		t.Errorf("expected response '42', got %q", fetched.Response)
	}
}

func TestUpdateAlreadyRespondedRequest(t *testing.T) {
	setupTestDB(t)

	req := Request{
		SourceName: "test-ide",
		AppName:    "app-3",
		Question:   "Yes or no?",
		Status:     "responded",
		Response:   "Yes",
	}
	instance.Create(&req)

	result := instance.Model(&Request{}).Where("id = ? AND status = ?", req.ID, "pending").Updates(map[string]interface{}{
		"response": "No",
		"status":   "responded",
	})
	if result.RowsAffected != 0 {
		t.Errorf("expected 0 rows affected for already-responded request, got %d", result.RowsAffected)
	}
}

func TestListRequestsOrderedByCreatedAt(t *testing.T) {
	setupTestDB(t)

	for i, q := range []string{"first", "second", "third"} {
		instance.Create(&Request{
			SourceName: "test-ide",
			AppName:    "app",
			Question:   q,
			Status:     "pending",
		})
		_ = i
	}

	var requests []Request
	if err := instance.Order("created_at DESC").Find(&requests).Error; err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(requests) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(requests))
	}
	if requests[0].Question != "third" {
		t.Errorf("expected newest first, got %q", requests[0].Question)
	}
}

func TestFilterByAppName(t *testing.T) {
	setupTestDB(t)

	instance.Create(&Request{SourceName: "test-ide", AppName: "alpha", Question: "q1", Status: "pending"})
	instance.Create(&Request{SourceName: "test-ide", AppName: "beta", Question: "q2", Status: "pending"})
	instance.Create(&Request{SourceName: "test-ide", AppName: "alpha", Question: "q3", Status: "pending"})

	var requests []Request
	instance.Where("app_name = ?", "alpha").Find(&requests)
	if len(requests) != 2 {
		t.Errorf("expected 2 alpha requests, got %d", len(requests))
	}
}
