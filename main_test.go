package main

import (
	"context"
	"encoding/json"
	"testing"

	"google.golang.org/api/calendar/v3"
)

// fakeCalendar implements CalendarService for testing
type fakeCalendar struct {
	events      []CalendarEvent
	err         error
	created     *calendar.Event
	updated     *calendar.Event
	lastDays    int
	lastStart   string
	lastEnd     string
	deletedID   string
	deleteErr   error
}

func (f *fakeCalendar) ListEventsForDays(_ context.Context, days int) ([]CalendarEvent, error) {
	f.lastDays = days
	return f.events, f.err
}

func (f *fakeCalendar) ListEventsRange(_ context.Context, start, end string) ([]CalendarEvent, error) {
	f.lastStart = start
	f.lastEnd = end
	return f.events, f.err
}

func (f *fakeCalendar) CreateEvent(_ context.Context, summary, description, date, startTime, endTime string) (*calendar.Event, error) {
	return f.created, f.err
}

func (f *fakeCalendar) UpdateEvent(_ context.Context, eventID string, updates EventUpdates) (*calendar.Event, error) {
	return f.updated, f.err
}

func (f *fakeCalendar) DeleteEvent(_ context.Context, eventID string) error {
	f.deletedID = eventID
	return f.deleteErr
}

func newTestServer(fake *fakeCalendar) *Server {
	return &Server{calendar: fake}
}

func TestHandleInitialize(t *testing.T) {
	s := newTestServer(&fakeCalendar{})
	req := JSONRPCRequest{JSONRPC: "2.0", ID: float64(1), Method: "initialize"}

	resp := s.handleRequest(req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("expected protocol version 2024-11-05, got %v", result["protocolVersion"])
	}

	serverInfo := result["serverInfo"].(map[string]string)
	if serverInfo["name"] != "google-calendar" {
		t.Errorf("expected server name google-calendar, got %s", serverInfo["name"])
	}
}

func TestHandleInitialized(t *testing.T) {
	s := newTestServer(&fakeCalendar{})
	req := JSONRPCRequest{JSONRPC: "2.0", Method: "initialized"}

	resp := s.handleRequest(req)
	if resp != nil {
		t.Error("expected nil response for initialized notification")
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	s := newTestServer(&fakeCalendar{})
	req := JSONRPCRequest{JSONRPC: "2.0", ID: float64(1), Method: "unknown/method"}

	resp := s.handleRequest(req)
	if resp == nil {
		t.Fatal("expected error response")
	}
	if resp.Error == nil {
		t.Fatal("expected RPC error")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestHandleToolsList(t *testing.T) {
	s := newTestServer(&fakeCalendar{})
	req := JSONRPCRequest{JSONRPC: "2.0", ID: float64(1), Method: "tools/list"}

	resp := s.handleRequest(req)
	if resp == nil {
		t.Fatal("expected response")
	}

	result := resp.Result.(map[string]interface{})
	tools := result["tools"].([]map[string]interface{})

	expectedTools := []string{"list_events", "list_events_range", "create_event", "delete_event", "edit_event"}
	if len(tools) != len(expectedTools) {
		t.Fatalf("expected %d tools, got %d", len(expectedTools), len(tools))
	}

	for i, name := range expectedTools {
		if tools[i]["name"] != name {
			t.Errorf("tool %d: expected name %q, got %q", i, name, tools[i]["name"])
		}
	}
}

func TestCallUnknownTool(t *testing.T) {
	s := newTestServer(&fakeCalendar{})
	params, _ := json.Marshal(map[string]interface{}{"name": "nonexistent"})
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "tools/call",
		Params:  params,
	}

	resp := s.handleRequest(req)
	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("expected error code -32602, got %d", resp.Error.Code)
	}
}

func TestCallListEvents_Default7Days(t *testing.T) {
	fake := &fakeCalendar{
		events: []CalendarEvent{
			{ID: "1", Summary: "Test Event", Start: "2026-02-20T10:00:00+04:00", End: "2026-02-20T11:00:00+04:00"},
		},
	}
	s := newTestServer(fake)

	resp := s.callListEvents(context.Background(), float64(1), nil)
	if fake.lastDays != 7 {
		t.Errorf("expected default 7 days, got %d", fake.lastDays)
	}

	result := resp.Result.(map[string]interface{})
	content := result["content"].([]map[string]string)
	if len(content) == 0 {
		t.Fatal("expected content in response")
	}
	text := content[0]["text"]
	if text == "" {
		t.Error("expected non-empty text")
	}
}

func TestCallListEvents_CustomDays(t *testing.T) {
	fake := &fakeCalendar{events: []CalendarEvent{}}
	s := newTestServer(fake)

	args, _ := json.Marshal(map[string]int{"days": 14})
	s.callListEvents(context.Background(), float64(1), args)

	if fake.lastDays != 14 {
		t.Errorf("expected 14 days, got %d", fake.lastDays)
	}
}

func TestCallListEvents_InvalidDaysUsesDefault(t *testing.T) {
	fake := &fakeCalendar{events: []CalendarEvent{}}
	s := newTestServer(fake)

	args, _ := json.Marshal(map[string]int{"days": -1})
	s.callListEvents(context.Background(), float64(1), args)

	if fake.lastDays != 7 {
		t.Errorf("expected default 7 days for negative input, got %d", fake.lastDays)
	}
}

func TestCallListEventsRange(t *testing.T) {
	fake := &fakeCalendar{
		events: []CalendarEvent{
			{ID: "1", Summary: "Range Event", Start: "2026-03-01", End: "2026-03-02"},
		},
	}
	s := newTestServer(fake)

	args, _ := json.Marshal(map[string]string{"start_date": "2026-03-01", "end_date": "2026-03-31"})
	resp := s.callListEventsRange(context.Background(), float64(1), args)

	if fake.lastStart != "2026-03-01" {
		t.Errorf("expected start 2026-03-01, got %s", fake.lastStart)
	}
	if fake.lastEnd != "2026-03-31" {
		t.Errorf("expected end 2026-03-31, got %s", fake.lastEnd)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}
}

func TestCallListEventsRange_MissingParams(t *testing.T) {
	s := newTestServer(&fakeCalendar{})

	args, _ := json.Marshal(map[string]string{"start_date": "2026-03-01"})
	resp := s.callListEventsRange(context.Background(), float64(1), args)

	if resp.Error == nil {
		t.Error("expected error for missing end_date")
	}
}

func TestCallCreateEvent(t *testing.T) {
	fake := &fakeCalendar{
		created: &calendar.Event{Id: "new-id", HtmlLink: "https://calendar.google.com/event/new-id"},
	}
	s := newTestServer(fake)

	args, _ := json.Marshal(map[string]string{
		"summary":    "New Event",
		"date":       "2026-03-15",
		"start_time": "10:00",
		"end_time":   "11:00",
	})
	resp := s.callCreateEvent(context.Background(), float64(1), args)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	result := resp.Result.(map[string]interface{})
	content := result["content"].([]map[string]string)
	text := content[0]["text"]
	if text == "" {
		t.Error("expected non-empty response text")
	}
}

func TestCallCreateEvent_MissingRequired(t *testing.T) {
	s := newTestServer(&fakeCalendar{})

	args, _ := json.Marshal(map[string]string{"summary": "No times"})
	resp := s.callCreateEvent(context.Background(), float64(1), args)

	if resp.Error == nil {
		t.Error("expected error for missing required fields")
	}
}

func TestCallDeleteEvent(t *testing.T) {
	fake := &fakeCalendar{}
	s := newTestServer(fake)

	args, _ := json.Marshal(map[string]string{"event_id": "evt-del"})
	resp := s.callDeleteEvent(context.Background(), float64(1), args)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if fake.deletedID != "evt-del" {
		t.Errorf("expected deleted ID evt-del, got %s", fake.deletedID)
	}
}

func TestCallDeleteEvent_MissingEventID(t *testing.T) {
	s := newTestServer(&fakeCalendar{})

	args, _ := json.Marshal(map[string]string{})
	resp := s.callDeleteEvent(context.Background(), float64(1), args)

	if resp.Error == nil {
		t.Error("expected error for missing event_id")
	}
}

func TestCallEditEvent(t *testing.T) {
	fake := &fakeCalendar{
		updated: &calendar.Event{Id: "evt-1", Summary: "Updated", HtmlLink: "https://calendar.google.com/event/evt-1"},
	}
	s := newTestServer(fake)

	summary := "Updated"
	args, _ := json.Marshal(map[string]interface{}{"event_id": "evt-1", "summary": summary})
	resp := s.callEditEvent(context.Background(), float64(1), args)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestCallEditEvent_MissingEventID(t *testing.T) {
	s := newTestServer(&fakeCalendar{})

	args, _ := json.Marshal(map[string]string{"summary": "No ID"})
	resp := s.callEditEvent(context.Background(), float64(1), args)

	if resp.Error == nil {
		t.Error("expected error for missing event_id")
	}
}

func TestSuccessResponse(t *testing.T) {
	s := newTestServer(&fakeCalendar{})
	resp := s.successResponse(float64(1), "hello")

	result := resp.Result.(map[string]interface{})
	content := result["content"].([]map[string]string)
	if content[0]["type"] != "text" {
		t.Errorf("expected type text, got %s", content[0]["type"])
	}
	if content[0]["text"] != "hello" {
		t.Errorf("expected text hello, got %s", content[0]["text"])
	}
}

func TestErrorResponse(t *testing.T) {
	s := newTestServer(&fakeCalendar{})
	resp := s.errorResponse(float64(1), context.DeadlineExceeded)

	result := resp.Result.(map[string]interface{})
	isError := result["isError"].(bool)
	if !isError {
		t.Error("expected isError to be true")
	}
}

func TestFormatEvents_Empty(t *testing.T) {
	s := newTestServer(&fakeCalendar{})
	text := s.formatEvents(nil)
	if text != "No events found." {
		t.Errorf("expected 'No events found.', got %q", text)
	}
}

func TestFormatEvents_Multiple(t *testing.T) {
	s := newTestServer(&fakeCalendar{})
	events := []CalendarEvent{
		{ID: "1", Summary: "First", Start: "2026-02-20T10:00:00+04:00", End: "2026-02-20T11:00:00+04:00"},
		{ID: "2", Summary: "Second", Start: "2026-02-21T14:00:00+04:00", End: "2026-02-21T15:00:00+04:00"},
	}
	text := s.formatEvents(events)

	if text == "No events found." {
		t.Error("should not return empty message for non-empty list")
	}
	// Check that both events are present
	for _, e := range events {
		if !contains(text, e.Summary) {
			t.Errorf("expected text to contain %q", e.Summary)
		}
		if !contains(text, e.ID) {
			t.Errorf("expected text to contain ID %q", e.ID)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
