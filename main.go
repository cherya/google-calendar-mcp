package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"google.golang.org/api/calendar/v3"
)

const (
	serverName    = "google-calendar"
	serverVersion = "1.0.0"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type CalendarService interface {
	ListEventsForDays(ctx context.Context, days int) ([]CalendarEvent, error)
	ListEventsRange(ctx context.Context, startDate, endDate string) ([]CalendarEvent, error)
	CreateEvent(ctx context.Context, summary, description, date, startTime, endTime string) (*calendar.Event, error)
	UpdateEvent(ctx context.Context, eventID string, updates EventUpdates) (*calendar.Event, error)
	DeleteEvent(ctx context.Context, eventID string) error
}

type Server struct {
	calendar CalendarService
}

func main() {
	credentialsFile := os.Getenv("GOOGLE_CREDENTIALS_FILE")
	calendarID := os.Getenv("CALENDAR_ID")

	if credentialsFile == "" || calendarID == "" {
		log.Fatal("GOOGLE_CREDENTIALS_FILE and CALENDAR_ID environment variables must be set")
	}

	cal, err := NewCalendarClient(credentialsFile, calendarID)
	if err != nil {
		log.Fatalf("Failed to create calendar client: %v", err)
	}

	server := &Server{calendar: cal}
	server.run()
}

func (s *Server) run() {
	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer size for large messages
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		response := s.handleRequest(req)
		if response != nil {
			s.sendResponse(response)
		}
	}
}

func (s *Server) sendResponse(resp *JSONRPCResponse) {
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.sendResponse(resp)
}

func (s *Server) handleRequest(req JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		return nil // notification, no response
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

func (s *Server) handleInitialize(req JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    serverName,
				"version": serverVersion,
			},
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
		},
	}
}

func (s *Server) handleToolsList(req JSONRPCRequest) *JSONRPCResponse {
	tools := []map[string]interface{}{
		{
			"name":        "list_events",
			"description": "List calendar events for the next N days",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"days": map[string]interface{}{
						"type":        "integer",
						"description": "Number of days to look ahead (default: 7)",
						"default":     7,
					},
				},
			},
		},
		{
			"name":        "list_events_range",
			"description": "List calendar events between two dates",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"start_date": map[string]interface{}{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD format",
					},
					"end_date": map[string]interface{}{
						"type":        "string",
						"description": "End date in YYYY-MM-DD format",
					},
				},
				"required": []string{"start_date", "end_date"},
			},
		},
		{
			"name":        "create_event",
			"description": "Create a new calendar event",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "Event title",
					},
					"date": map[string]interface{}{
						"type":        "string",
						"description": "Event date in YYYY-MM-DD format",
					},
					"start_time": map[string]interface{}{
						"type":        "string",
						"description": "Start time in HH:MM format (24-hour)",
					},
					"end_time": map[string]interface{}{
						"type":        "string",
						"description": "End time in HH:MM format (24-hour)",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Event description (optional)",
					},
				},
				"required": []string{"summary", "date", "start_time", "end_time"},
			},
		},
		{
			"name":        "delete_event",
			"description": "Delete a calendar event",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"event_id": map[string]interface{}{
						"type":        "string",
						"description": "Event ID to delete (from list_events)",
					},
				},
				"required": []string{"event_id"},
			},
		},
		{
			"name":        "edit_event",
			"description": "Edit an existing calendar event",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"event_id": map[string]interface{}{
						"type":        "string",
						"description": "Event ID to edit (from list_events)",
					},
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "New event title (optional)",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "New event description (optional)",
					},
					"date": map[string]interface{}{
						"type":        "string",
						"description": "New date in YYYY-MM-DD format (optional)",
					},
					"start_time": map[string]interface{}{
						"type":        "string",
						"description": "New start time in HH:MM format (optional)",
					},
					"end_time": map[string]interface{}{
						"type":        "string",
						"description": "New end time in HH:MM format (optional)",
					},
				},
				"required": []string{"event_id"},
			},
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func (s *Server) handleToolsCall(req JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid params",
				Data:    err.Error(),
			},
		}
	}

	ctx := context.Background()

	switch params.Name {
	case "list_events":
		return s.callListEvents(ctx, req.ID, params.Arguments)
	case "list_events_range":
		return s.callListEventsRange(ctx, req.ID, params.Arguments)
	case "create_event":
		return s.callCreateEvent(ctx, req.ID, params.Arguments)
	case "delete_event":
		return s.callDeleteEvent(ctx, req.ID, params.Arguments)
	case "edit_event":
		return s.callEditEvent(ctx, req.ID, params.Arguments)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32602,
				Message: "Unknown tool: " + params.Name,
			},
		}
	}
}

func (s *Server) callListEvents(ctx context.Context, id interface{}, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		Days int `json:"days"`
	}
	input.Days = 7 // default

	if len(args) > 0 {
		json.Unmarshal(args, &input)
	}

	if input.Days <= 0 {
		input.Days = 7
	}

	events, err := s.calendar.ListEventsForDays(ctx, input.Days)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.successResponse(id, s.formatEvents(events))
}

func (s *Server) callListEventsRange(ctx context.Context, id interface{}, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}

	if err := json.Unmarshal(args, &input); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid arguments",
				Data:    err.Error(),
			},
		}
	}

	if input.StartDate == "" || input.EndDate == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &RPCError{
				Code:    -32602,
				Message: "start_date and end_date are required",
			},
		}
	}

	events, err := s.calendar.ListEventsRange(ctx, input.StartDate, input.EndDate)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.successResponse(id, s.formatEvents(events))
}

func (s *Server) callCreateEvent(ctx context.Context, id interface{}, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		Summary     string `json:"summary"`
		Date        string `json:"date"`
		StartTime   string `json:"start_time"`
		EndTime     string `json:"end_time"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(args, &input); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid arguments",
				Data:    err.Error(),
			},
		}
	}

	if input.Summary == "" || input.Date == "" || input.StartTime == "" || input.EndTime == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &RPCError{
				Code:    -32602,
				Message: "summary, date, start_time, and end_time are required",
			},
		}
	}

	event, err := s.calendar.CreateEvent(ctx, input.Summary, input.Description, input.Date, input.StartTime, input.EndTime)
	if err != nil {
		return s.errorResponse(id, err)
	}

	result := fmt.Sprintf("Event created successfully!\nID: %s\nLink: %s", event.Id, event.HtmlLink)

	return s.successResponse(id, result)
}

func (s *Server) callDeleteEvent(ctx context.Context, id interface{}, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		EventID string `json:"event_id"`
	}

	if err := json.Unmarshal(args, &input); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid arguments",
				Data:    err.Error(),
			},
		}
	}

	if input.EventID == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &RPCError{
				Code:    -32602,
				Message: "event_id is required",
			},
		}
	}

	if err := s.calendar.DeleteEvent(ctx, input.EventID); err != nil {
		return s.errorResponse(id, err)
	}

	return s.successResponse(id, "Event deleted successfully!")
}

func (s *Server) callEditEvent(ctx context.Context, id interface{}, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		EventID     string  `json:"event_id"`
		Summary     *string `json:"summary"`
		Description *string `json:"description"`
		Date        *string `json:"date"`
		StartTime   *string `json:"start_time"`
		EndTime     *string `json:"end_time"`
	}

	if err := json.Unmarshal(args, &input); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid arguments",
				Data:    err.Error(),
			},
		}
	}

	if input.EventID == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &RPCError{
				Code:    -32602,
				Message: "event_id is required",
			},
		}
	}

	updates := EventUpdates{
		Summary:     input.Summary,
		Description: input.Description,
		Date:        input.Date,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
	}

	event, err := s.calendar.UpdateEvent(ctx, input.EventID, updates)
	if err != nil {
		return s.errorResponse(id, err)
	}

	result := fmt.Sprintf("Event updated successfully!\nID: %s\nSummary: %s\nLink: %s", event.Id, event.Summary, event.HtmlLink)

	return s.successResponse(id, result)
}

func (s *Server) formatEvents(events []CalendarEvent) string {
	if len(events) == 0 {
		return "No events found."
	}

	result := fmt.Sprintf("Found %d event(s):\n\n", len(events))
	for _, e := range events {
		result += fmt.Sprintf("- %s\n  Start: %s\n  End: %s\n  ID: %s\n\n", e.Summary, e.Start, e.End, e.ID)
	}

	return result
}

func (s *Server) successResponse(id interface{}, text string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": text},
			},
		},
	}
}

func (s *Server) errorResponse(id interface{}, err error) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": fmt.Sprintf("Error: %v", err)},
			},
			"isError": true,
		},
	}
}
